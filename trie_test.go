package cidranger

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	rnet "github.com/yl2chen/cidranger/net"
)

func TestPrefixTrieInsert(t *testing.T) {
	cases := []struct {
		version                      rnet.IPVersion
		inserts                      []string
		expectedNetworksInDepthOrder []string
		name                         string
	}{
		{rnet.IPv4, []string{"192.168.0.1/24"}, []string{"192.168.0.1/24"}, "basic insert"},
		{
			rnet.IPv4,
			[]string{"1.2.3.4/32", "1.2.3.5/32"},
			[]string{"1.2.3.4/32", "1.2.3.5/32"},
			"single ip IPv4 network insert",
		},
		{
			rnet.IPv6,
			[]string{"0::1/128", "0::2/128"},
			[]string{"0::1/128", "0::2/128"},
			"single ip IPv6 network insert",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/16", "192.168.0.1/24"},
			[]string{"192.168.0.1/16", "192.168.0.1/24"},
			"in order insert",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/32", "192.168.0.1/32"},
			[]string{"192.168.0.1/32"},
			"duplicate network insert",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.0.1/16"},
			[]string{"192.168.0.1/16", "192.168.0.1/24"},
			"reverse insert",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.1.1/24"},
			[]string{"192.168.0.1/24", "192.168.1.1/24"},
			"branch insert",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.1.1/24", "192.168.1.1/30"},
			[]string{"192.168.0.1/24", "192.168.1.1/24", "192.168.1.1/30"},
			"branch inserts",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version).(*prefixTrie)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			walk := trie.walkDepth()
			for _, network := range tc.expectedNetworksInDepthOrder {
				_, ipnet, _ := net.ParseCIDR(network)
				expected := NewBasicRangerEntry(*ipnet)
				actual := <-walk
				assert.Equal(t, expected, actual)
			}

			// Ensure no unexpected elements in trie.
			for network := range walk {
				assert.Nil(t, network)
			}
		})
	}
}

func TestPrefixTrieString(t *testing.T) {
	inserts := []string{"192.168.0.1/24", "192.168.1.1/24", "192.168.1.1/30"}
	trie := newPrefixTree(rnet.IPv4).(*prefixTrie)
	for _, insert := range inserts {
		_, network, _ := net.ParseCIDR(insert)
		trie.Insert(NewBasicRangerEntry(*network))
	}
	expected := `0.0.0.0/0 (target_pos:31:has_entry:false)
| 1--> 192.168.0.0/23 (target_pos:8:has_entry:false)
| | 0--> 192.168.0.0/24 (target_pos:7:has_entry:true)
| | 1--> 192.168.1.0/24 (target_pos:7:has_entry:true)
| | | 0--> 192.168.1.0/30 (target_pos:1:has_entry:true)`
	assert.Equal(t, expected, trie.String())
}

func TestPrefixTrieRemove(t *testing.T) {
	cases := []struct {
		version                      rnet.IPVersion
		inserts                      []string
		removes                      []string
		expectedRemoves              []string
		expectedNetworksInDepthOrder []string
		name                         string
	}{
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24"},
			[]string{"192.168.0.1/24"},
			[]string{"192.168.0.1/24"},
			[]string{},
			"basic remove",
		},
		{
			rnet.IPv4,
			[]string{"1.2.3.4/32", "1.2.3.5/32"},
			[]string{"1.2.3.5/32"},
			[]string{"1.2.3.5/32"},
			[]string{"1.2.3.4/32"},
			"single ip IPv4 network remove",
		},
		{
			rnet.IPv4,
			[]string{"0::1/128", "0::2/128"},
			[]string{"0::2/128"},
			[]string{"0::2/128"},
			[]string{"0::1/128"},
			"single ip IPv6 network remove",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.0.1/25", "192.168.0.1/26"},
			[]string{"192.168.0.1/25"},
			[]string{"192.168.0.1/25"},
			[]string{"192.168.0.1/24", "192.168.0.1/26"},
			"remove path prefix",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.0.1/25", "192.168.0.64/26", "192.168.0.1/26"},
			[]string{"192.168.0.1/25"},
			[]string{"192.168.0.1/25"},
			[]string{"192.168.0.1/24", "192.168.0.1/26", "192.168.0.64/26"},
			"remove path prefix with more than 1 children",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.1/24", "192.168.0.1/25"},
			[]string{"192.168.0.1/26"},
			[]string{""},
			[]string{"192.168.0.1/24", "192.168.0.1/25"},
			"remove non existent",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version).(*prefixTrie)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			for i, remove := range tc.removes {
				_, network, _ := net.ParseCIDR(remove)
				removed, err := trie.Remove(*network)
				assert.NoError(t, err)
				if str := tc.expectedRemoves[i]; str != "" {
					_, ipnet, _ := net.ParseCIDR(str)
					expected := NewBasicRangerEntry(*ipnet)
					assert.Equal(t, expected, removed)
				} else {
					assert.Nil(t, removed)
				}
			}
			walk := trie.walkDepth()
			for _, network := range tc.expectedNetworksInDepthOrder {
				_, ipnet, _ := net.ParseCIDR(network)
				expected := NewBasicRangerEntry(*ipnet)
				actual := <-walk
				assert.Equal(t, expected, actual)
			}

			// Ensure no unexpected elements in trie.
			for network := range walk {
				assert.Nil(t, network)
			}
		})
	}
}

func TestToReplicateIssue(t *testing.T) {
	cases := []struct {
		version  rnet.IPVersion
		inserts  []string
		ip       net.IP
		networks []string
		name     string
	}{
		{
			rnet.IPv4,
			[]string{"192.168.0.1/32"},
			net.ParseIP("192.168.0.1"),
			[]string{"192.168.0.1/32"},
			"basic containing network for /32 mask",
		},
		{
			rnet.IPv6,
			[]string{"a::1/128"},
			net.ParseIP("a::1"),
			[]string{"a::1/128"},
			"basic containing network for /128 mask",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			expectedEntries := []RangerEntry{}
			for _, network := range tc.networks {
				_, net, _ := net.ParseCIDR(network)
				expectedEntries = append(expectedEntries, NewBasicRangerEntry(*net))
			}
			contains, err := trie.Contains(tc.ip)
			assert.NoError(t, err)
			assert.True(t, contains)
			networks, err := trie.ContainingNetworks(tc.ip)
			assert.NoError(t, err)
			assert.Equal(t, expectedEntries, networks)
		})
	}
}

type expectedIPRange struct {
	start net.IP
	end   net.IP
}

func TestPrefixTrieContains(t *testing.T) {
	cases := []struct {
		version     rnet.IPVersion
		inserts     []string
		expectedIPs []expectedIPRange
		name        string
	}{
		{
			rnet.IPv4,
			[]string{"192.168.0.0/24"},
			[]expectedIPRange{
				{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.1.0")},
			},
			"basic contains",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.0/24", "128.168.0.0/24"},
			[]expectedIPRange{
				{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.1.0")},
				{net.ParseIP("128.168.0.0"), net.ParseIP("128.168.1.0")},
			},
			"multiple ranges contains",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			for _, expectedIPRange := range tc.expectedIPs {
				var contains bool
				var err error
				start := expectedIPRange.start
				for ; !expectedIPRange.end.Equal(start); start = rnet.NextIP(start) {
					contains, err = trie.Contains(start)
					assert.NoError(t, err)
					assert.True(t, contains)
				}

				// Check out of bounds ips on both ends
				contains, err = trie.Contains(rnet.PreviousIP(expectedIPRange.start))
				assert.NoError(t, err)
				assert.False(t, contains)
				contains, err = trie.Contains(rnet.NextIP(expectedIPRange.end))
				assert.NoError(t, err)
				assert.False(t, contains)
			}
		})
	}
}

func TestPrefixTrieContainingNetworks(t *testing.T) {
	cases := []struct {
		version  rnet.IPVersion
		inserts  []string
		ip       net.IP
		networks []string
		name     string
	}{
		{
			rnet.IPv4,
			[]string{"192.168.0.0/24"},
			net.ParseIP("192.168.0.1"),
			[]string{"192.168.0.0/24"},
			"basic containing networks",
		},
		{
			rnet.IPv4,
			[]string{"192.168.0.0/24", "192.168.0.0/25"},
			net.ParseIP("192.168.0.1"),
			[]string{"192.168.0.0/24", "192.168.0.0/25"},
			"inclusive networks",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			expectedEntries := []RangerEntry{}
			for _, network := range tc.networks {
				_, net, _ := net.ParseCIDR(network)
				expectedEntries = append(expectedEntries, NewBasicRangerEntry(*net))
			}
			networks, err := trie.ContainingNetworks(tc.ip)
			assert.NoError(t, err)
			assert.Equal(t, expectedEntries, networks)
		})
	}
}

type coveredNetworkTest struct {
	version  rnet.IPVersion
	inserts  []string
	search   string
	networks []string
	name     string
}

var coveredNetworkTests = []coveredNetworkTest{
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24"},
		"192.168.0.0/16",
		[]string{"192.168.0.0/24"},
		"basic covered networks",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24"},
		"10.1.0.0/16",
		nil,
		"nothing",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24", "192.168.0.0/25"},
		"192.168.0.0/16",
		[]string{"192.168.0.0/24", "192.168.0.0/25"},
		"multiple networks",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24", "192.168.0.0/25", "192.168.0.1/32"},
		"192.168.0.0/16",
		[]string{"192.168.0.0/24", "192.168.0.0/25", "192.168.0.1/32"},
		"multiple networks 2",
	},
	{
		rnet.IPv4,
		[]string{"192.168.1.1/32"},
		"192.168.0.0/16",
		[]string{"192.168.1.1/32"},
		"leaf",
	},
	{
		rnet.IPv4,
		[]string{"0.0.0.0/0", "192.168.1.1/32"},
		"192.168.0.0/16",
		[]string{"192.168.1.1/32"},
		"leaf with root",
	},
	{
		rnet.IPv4,
		[]string{
			"0.0.0.0/0", "192.168.0.0/24", "192.168.1.1/32",
			"10.1.0.0/16", "10.1.1.0/24",
		},
		"192.168.0.0/16",
		[]string{"192.168.0.0/24", "192.168.1.1/32"},
		"path not taken",
	},
	{
		rnet.IPv4,
		[]string{
			"192.168.0.0/15",
		},
		"192.168.0.0/16",
		nil,
		"only masks different",
	},
}

func TestPrefixTrieCoveredNetworks(t *testing.T) {
	for _, tc := range coveredNetworkTests {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			var expectedEntries []RangerEntry
			for _, network := range tc.networks {
				_, net, _ := net.ParseCIDR(network)
				expectedEntries = append(expectedEntries,
					NewBasicRangerEntry(*net))
			}
			_, snet, _ := net.ParseCIDR(tc.search)
			networks, err := trie.CoveredNetworks(*snet)
			assert.NoError(t, err)
			assert.Equal(t, expectedEntries, networks)
		})
	}
}

var coveringNetworkTests = []coveredNetworkTest{
	{
		rnet.IPv4,
		[]string{"192.168.0.0/16"},
		"192.168.0.0/24",
		[]string{"192.168.0.0/16"},
		"basic covering networks (only masks different)",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24"},
		"192.168.0.0/24",
		[]string{"192.168.0.0/24"},
		"covering the same networks",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24"},
		"10.1.0.0/16",
		nil,
		"nothing",
	},
	{
		rnet.IPv4,
		[]string{"192.168.1.0/24", "192.168.2.0/25"},
		"192.168.3.0/24",
		nil,
		"nothing 2 (has similar prefix)",
	},
	{
		rnet.IPv4,
		[]string{"192.168.1.0/24", "192.168.2.0/24"},
		"192.168.0.0/16",
		nil,
		"nothing 3 (larger range)",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24", "192.168.0.0/25"},
		"192.168.0.0/25",
		[]string{"192.168.0.0/24", "192.168.0.0/25"},
		"multiple networks",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/24", "192.168.0.0/25", "192.168.0.1/32"},
		"192.168.0.1/32",
		[]string{"192.168.0.0/24", "192.168.0.0/25", "192.168.0.1/32"},
		"multiple networks with leaf",
	},
	{
		rnet.IPv4,
		[]string{"192.168.0.0/16"},
		"192.168.1.1/32",
		[]string{"192.168.0.0/16"},
		"leaf with the same network",
	},
	{
		rnet.IPv4,
		[]string{"0.0.0.0/0", "192.168.1.1/32"},
		"192.168.0.0/16",
		[]string{"0.0.0.0/0"},
		"leaf with root",
	},
	{
		rnet.IPv4,
		[]string{
			"192.168.0.0/24", "192.168.1.1/32",
			"10.1.0.0/16", "10.1.1.0/24",
		},
		"192.168.0.0/16",
		[]string{},
		"path not taken",
	},
}

func TestPrefixTrieCoveringNetworks(t *testing.T) {
	for _, tc := range coveringNetworkTests {
		t.Run(tc.name, func(t *testing.T) {
			trie := newPrefixTree(tc.version)
			for _, insert := range tc.inserts {
				_, network, _ := net.ParseCIDR(insert)
				err := trie.Insert(NewBasicRangerEntry(*network))
				assert.NoError(t, err)
			}
			var expectedEntries []RangerEntry
			for _, network := range tc.networks {
				_, net, _ := net.ParseCIDR(network)
				expectedEntries = append(expectedEntries,
					NewBasicRangerEntry(*net))
			}
			_, snet, _ := net.ParseCIDR(tc.search)
			networks, err := trie.CoveringNetworks(*snet)
			assert.NoError(t, err)
			assert.Equal(t, expectedEntries, networks)
		})
	}
}

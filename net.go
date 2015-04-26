package critbitgo

import (
	"net"
)

// IP routing table.
type Net struct {
	trie *Trie
}

// Associates value with `s`.
// If `s` is not CIDR notation, returns an error.
func (n *Net) AddCIDR(s string, value interface{}) error {
	key, err := netCidrToKey(s)
	if err != nil {
		return err
	}
	return n.trie.Set(key, value)
}

// Deletes the mapping for `s`.
// If `s` is not CIDR notation or the mapping is not found, return false.
func (n *Net) DeleteCIDR(s string) bool {
	key, err := netCidrToKey(s)
	if err != nil {
		return false
	}
	return n.trie.Delete(key)
}

// Returns the value to which `s` is mapped.
// If `s` is not CIDR notation, returns an error.
func (n *Net) GetCIDR(s string) (value interface{}, err error) {
	key, err := netCidrToKey(s)
	if err == nil {
		if node := n.trie.search(key); node.external != nil {
			value = node.external.value
		}
	}
	return
}

// Returns the value by using the longest prefix matching.
// If `s` is not CIDR notation, returns an error.
func (n *Net) MatchCIDR(s string) (cidr string, value interface{}, err error) {
	key, err := netCidrToKey(s)
	if err != nil || n.trie.size == 0 {
		return
	}
	if node := match(&n.trie.root, key, false); node != nil {
		cidr = netKeyToCidr(node.external.key)
		value = node.external.value
	}
	return
}

func match(p *node, key []byte, backtracking bool) *node {
	if p.internal != nil {
		var direction int
		if p.internal.offset == len(key)-2 {
			// selecting the larger side when comparing the mask
			direction = 1
		} else if backtracking {
			direction = 0
		} else {
			direction = p.internal.direction(key)
		}

		if c := match(&p.internal.child[direction], key, backtracking); c != nil {
			return c
		}
		if direction == 1 {
			// search other node
			return match(&p.internal.child[0], key, true)
		}
		return nil
	} else {
		nlen := len(p.external.key)
		if nlen != len(key) {
			return nil
		}

		// check mask
		mask := p.external.key[nlen-2]
		if mask > key[nlen-2] {
			return nil
		}

		// compare both keys with mask
		div := int(mask >> 3)
		for i := 0; i < div; i++ {
			if p.external.key[i] != key[i] {
				return nil
			}
		}
		if mod := uint(mask & 0x07); mod > 0 {
			bit := 8 - mod
			if p.external.key[div] != key[div]&(0xff>>bit<<bit) {
				return nil
			}
		}
		return p
	}
}

// Deletes all mappings
func (n *Net) Clear() {
	n.trie.Clear()
}

// Returns number of mappings
func (n *Net) Size() int {
	return n.trie.Size()
}

// Create IP routing table
func NewNet() *Net {
	return &Net{NewTrie()}
}

// Create IP routing table with the specified initial capacity.
func NewNetWithCapacity(c int) *Net {
	return &Net{NewTrieWithCapacity(c)}
}

func netCidrToKey(s string) ([]byte, error) {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	ones, _ := ipnet.Mask.Size()
	// +--------------+------+--------------+
	// | ip address.. | mask | termination  |
	// +--------------+------+--------------+
	return append(append(ipnet.IP, byte(ones)), 0xff), nil
}

func netKeyToCidr(k []byte) string {
	iplen := len(k) - 2
	ipnet := &net.IPNet{
		IP:   net.IP(k[:iplen]),
		Mask: net.CIDRMask(int(k[iplen]), iplen*8),
	}
	return ipnet.String()
}

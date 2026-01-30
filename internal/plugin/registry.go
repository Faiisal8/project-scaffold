package plugin

import "sync"

var (
	mu      sync.RWMutex
	plugins = make(map[string]Plugin)
)

func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	plugins[p.Name()] = p
}

func Get(name string) Plugin {
	mu.RLock()
	defer mu.RUnlock()
	return plugins[name]
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(plugins))
	for n := range plugins {
		names = append(names, n)
	}
	return names
}

func CompatibleWith(stackKey string) []string {
	mu.RLock()
	defer mu.RUnlock()
	var out []string
	for name, p := range plugins {
		for _, s := range p.CompatibleStacks() {
			if s == stackKey {
				out = append(out, name)
				break
			}
		}
	}
	return out
}

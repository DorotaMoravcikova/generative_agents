package memory

type Worlds map[string]map[string]map[string]map[string]struct{}

type Spatial struct {
	// world->sector->arena->object
	worlds Worlds
}

func NewSpatial() *Spatial {
	return &Spatial{
		worlds: make(Worlds),
	}
}

func (store *Spatial) Worlds() Worlds { return store.worlds }

func (store *Spatial) Register(path Path) {
	if path.world != "" {
		if _, ok := store.worlds[path.world]; !ok {
			store.worlds[path.world] = map[string]map[string]map[string]struct{}{}
		}
	}
	if path.sector != "" {
		if _, ok := store.worlds[path.world][path.sector]; !ok {
			store.worlds[path.world][path.sector] = map[string]map[string]struct{}{}
		}
	}
	if path.arena != "" {
		if _, ok := store.worlds[path.world][path.sector][path.arena]; !ok {
			store.worlds[path.world][path.sector][path.arena] = map[string]struct{}{}
		}
	}
	if path.object != "" {
		if _, ok := store.worlds[path.world][path.sector][path.arena][path.object]; !ok {
			store.worlds[path.world][path.sector][path.arena][path.object] = struct{}{}
		}
	}
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func (store *Spatial) GetKnown(path Path, level PathLevel) []string {
	if level == PathLevelInvalid {
		return []string{}
	}
	if level <= PathLevelWorld {
		return keys(store.worlds)
	}

	sectors, ok := store.worlds[path.Get(PathLevelWorld)]
	if !ok {
		return []string{}
	} else if level <= PathLevelSector {
		return keys(sectors)
	}

	arenas, ok := sectors[path.Get(PathLevelSector)]
	if !ok {
		return []string{}
	} else if level <= PathLevelArena {
		return keys(arenas)
	}

	objects, ok := arenas[path.Get(PathLevelArena)]
	if !ok {
		return []string{}
	}
	return keys(objects)
}

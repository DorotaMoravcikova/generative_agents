package maze

import (
	"fmt"
	"maps"
	"math"

	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type Event struct {
	SPO         memory.SPO
	Description string
}

type TilePos struct {
	X, Y int
}

func (t TilePos) EuclidianDistance(o TilePos) float64 {
	return math.Sqrt(float64(
		(t.X-o.X)*(t.X-o.X) +
			(t.Y-o.Y)*(t.Y-o.Y)))
}

type Tile struct {
	Path             memory.Path
	SpawningLocation string
	Collision        bool
	Events           map[Event]struct{}
}

type Maze struct {
	name   string
	folder string

	width  int
	height int

	tileSize int

	// A HeightxWidth array representing all the tiles that are impossible to stand on.
	collisionInfo [][]bool
	// A HeightxWidth
	tiles [][]Tile
	// Maps a path to all tiles that correspond to that path
	addressTiles map[memory.Path][]TilePos
}

func (m Maze) Name() string {
	return m.name
}

func (m Maze) Folder() string {
	return m.folder
}

func (m *Maze) PathToTiles(plan memory.Path) ([]TilePos, bool) {
	t, ok := m.addressTiles[plan]
	return t, ok
}

func New(name, folder string, width, height int, tileSize int, collisionInfo [][]bool, tiles [][]Tile) *Maze {
	if len(tiles) != height {
		panic("tiles length does not match specified maze height")
	}
	for i, row := range tiles {
		if len(row) != width {
			panic(fmt.Errorf("tiles row %d width does not match specified maze width", i))
		}
	}

	if len(collisionInfo) != height {
		panic("collision info length does not match specified maze height")
	}
	for i, row := range collisionInfo {
		if len(row) != width {
			panic(fmt.Errorf("collision info row %d width does not match specified maze width", i))
		}
	}

	addressTiles := map[memory.Path][]TilePos{}

	for i := range tiles {
		for j, tile := range tiles[i] {
			addresses := []memory.Path{}

			level := tile.Path.Level()
			if level >= memory.PathLevelSector {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelSector))
			}
			if level >= memory.PathLevelArena {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelArena))
			}
			if level >= memory.PathLevelObject {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelObject))
			}

			for _, a := range addresses {
				addressTiles[a] = append(addressTiles[a], TilePos{Y: i, X: j})
			}
		}
	}

	return &Maze{
		name,
		folder,
		width, height,
		tileSize,
		collisionInfo, tiles,
		addressTiles,
	}
}

func (m *Maze) Exists(p memory.Path) bool {
	_, ok := m.addressTiles[p]

	return ok
}

func (m *Maze) GetTile(pos TilePos) Tile {
	return m.tiles[pos.Y][pos.X]
}

func (m *Maze) UpdateTile(pos TilePos, f func(*Tile)) {
	f(&m.tiles[pos.Y][pos.X])
}

func (m *Maze) GetNearbyTiles(tile TilePos, visionRadius int) []TilePos {
	left := 0
	right := m.width
	top := 0
	bottom := m.height

	// The +1s here are so we get a square with pos in the middle
	if tile.X-visionRadius > left {
		left = tile.X - visionRadius
	}
	if tile.X+visionRadius+1 < right {
		right = tile.X + visionRadius + 1
	}
	if tile.Y-visionRadius > top {
		top = tile.Y - visionRadius
	}
	if tile.Y+visionRadius+1 < bottom {
		bottom = tile.Y + visionRadius + 1
	}

	visionDiameter := 2*visionRadius + 1
	nearby := make([]TilePos, 0, visionDiameter*visionDiameter)
	for x := left; x < right; x++ {
		for y := top; y < bottom; y++ {
			nearby = append(nearby, TilePos{x, y})
		}
	}
	return nearby
}

func (m *Maze) AddEventToTile(tile TilePos, event Event) {
	m.tiles[tile.Y][tile.X].Events[event] = struct{}{}
}

func (m *Maze) RemoveEventFromTile(tile TilePos, event Event) {
	delete(m.tiles[tile.Y][tile.X].Events, event)
}

func (m *Maze) RemoveSubjectEventsFromTile(tile TilePos, subject string) {
	m.UpdateTile(tile, func(t *Tile) {
		maps.DeleteFunc(t.Events, func(ev Event, _ struct{}) bool {
			return ev.SPO.Subject == subject
		})
	})
}

func (m *Maze) TurnTileEventIdle(tile TilePos, ev Event) {
	m.UpdateTile(tile, func(t *Tile) {
		delete(t.Events, ev)
		t.Events[Event{SPO: memory.SPO{Subject: ev.SPO.Subject}}] = struct{}{}
	})
}

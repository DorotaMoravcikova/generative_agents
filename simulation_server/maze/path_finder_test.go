package maze_test

import (
	"testing"

	"github.com/fvdveen/generative_agents/simulation_server/maze"
)

func makeMaze() *maze.Maze {
	height := 8
	width := 13
	mazeRepr := [][]byte{
		{'#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#'},
		{' ', ' ', '#', ' ', ' ', ' ', ' ', ' ', '#', ' ', ' ', ' ', '#'},
		{'#', ' ', '#', ' ', ' ', '#', '#', ' ', ' ', ' ', '#', ' ', '#'},
		{'#', ' ', '#', ' ', ' ', '#', '#', ' ', '#', ' ', '#', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#', ' ', ' ', ' ', '#'},
		{'#', '#', '#', ' ', '#', ' ', '#', '#', '#', ' ', '#', ' ', '#'},
		{'#', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ', '#', ' ', ' '},
		{'#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#', '#'},
	}

	collision := make([][]bool, height)
	tiles := make([][]maze.Tile, height)

	for i := 0; i < height; i += 1 {
		for j := 0; j < width; j += 1 {
			tiles[i] = append(tiles[i], maze.Tile{})
			c := false
			if mazeRepr[i][j] == '#' {
				c = true
			}
			collision[i] = append(collision[i], c)
		}
	}

	return maze.New("", "", width, height, 1, collision, tiles)
}

func TestSameSquare(t *testing.T) {
	m := makeMaze()

	pos := maze.TilePos{Y: 0, X: 1}

	path := m.Pathfind(pos, pos)

	if len(path) != 1 && path[0] != pos {
		t.Fatalf("Wrond path: %v, expected [(0, 1)]", path)
	}
}

func TestOppositeSideOfMap(t *testing.T) {
	m := makeMaze()

	start := maze.TilePos{Y: 1, X: 0}
	end := maze.TilePos{Y: 6, X: 12}

	path := m.Pathfind(start, end)
	expected := []maze.TilePos{
		{X: 0, Y: 1},
		{X: 1, Y: 1},
		{X: 1, Y: 2},
		{X: 1, Y: 3},
		{X: 1, Y: 4},
		{X: 2, Y: 4},
		{X: 3, Y: 4},
		{X: 4, Y: 4},
		{X: 5, Y: 4},
		{X: 6, Y: 4},
		{X: 7, Y: 4},
		{X: 7, Y: 3},
		{X: 7, Y: 2},
		{X: 8, Y: 2},
		{X: 9, Y: 2},
		{X: 9, Y: 3},
		{X: 9, Y: 4},
		{X: 10, Y: 4},
		{X: 11, Y: 4},
		{X: 11, Y: 5},
		{X: 11, Y: 6},
		{X: 12, Y: 6},
	}

	if len(path) != len(expected) {
		t.Fatalf("Expected path length differs, got: %d, want: %d", len(path), len(expected))
	}

	for i := range path {
		if path[i] != expected[i] {
			t.Fatalf("Wrong path element at index %d, got: %v, want: %v", i, path[i], expected[i])
		}
	}
}

package maze

import "slices"

func (m *Maze) Pathfind(start, end TilePos) []TilePos {
	makeStep := func(d [][]int, k int) {
		for i := 0; i < len(d); i += 1 {
			for j := 0; j < len(d[i]); j += 1 {
				if d[i][j] != k {
					continue
				}
				if i > 0 && d[i-1][j] == 0 && !m.collisionInfo[i-1][j] {
					d[i-1][j] = k + 1
				}
				if j > 0 && d[i][j-1] == 0 && !m.collisionInfo[i][j-1] {
					d[i][j-1] = k + 1
				}
				if i < len(d)-1 && d[i+1][j] == 0 && !m.collisionInfo[i+1][j] {
					d[i+1][j] = k + 1
				}
				if j < len(d[i])-1 && d[i][j+1] == 0 && !m.collisionInfo[i][j+1] {
					d[i][j+1] = k + 1
				}
			}
		}
	}

	// NOTE(Friso): Remember the maze is height*width so we mirror that here
	distMaze := make([][]int, 0, len(m.tiles))
	for i := range m.tiles {
		distMaze = append(distMaze, make([]int, len(m.tiles[i])))
	}

	// NOTE(Friso): Remember the maze is height*width
	distMaze[start.Y][start.X] = 1

	k := 0
	loopMax := 150

	// NOTE(Friso): Remember the maze is height*width
	for distMaze[end.Y][end.X] == 0 && loopMax > 0 {
		k += 1
		makeStep(distMaze, k)

		loopMax -= 1
	}

	// NOTE(Friso): Remember the maze is height*width
	i, j := end.Y, end.X
	k = distMaze[i][j]
	path := append(make([]TilePos, 0, k), end)
	for k > 1 {
		if i > 0 && distMaze[i-1][j] == k-1 {
			i = i - 1
			path = append(path, TilePos{Y: i, X: j})
			k -= 1
		} else if j > 0 && distMaze[i][j-1] == k-1 {
			j = j - 1
			path = append(path, TilePos{Y: i, X: j})
			k -= 1
		} else if i < len(distMaze)-1 && distMaze[i+1][j] == k-1 {
			i = i + 1
			path = append(path, TilePos{Y: i, X: j})
			k -= 1
		} else if j < len(distMaze[i])-1 && distMaze[i][j+1] == k-1 {
			j = j + 1
			path = append(path, TilePos{Y: i, X: j})
			k -= 1
		}
	}

	slices.Reverse(path)

	return path
}

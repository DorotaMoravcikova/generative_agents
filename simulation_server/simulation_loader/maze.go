package simulationloader

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func readCSVFile(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(r); i += 1 {
		for j := 0; j < len(r[0]); j += 1 {
			r[i][j] = strings.TrimSpace(r[i][j])
		}
	}

	return r, nil
}

func LoadMaze(mazePath string, mazeName string) (*maze.Maze, error) {
	matrixFolder := path.Join(mazePath, "matrix")
	metaFile := path.Join(matrixFolder, "maze_meta_info.json")

	content, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, fmt.Errorf("could not read meta file %s: %w", mazePath, err)
	}

	var meta MazeMetaInfo
	if err = json.Unmarshal(content, &meta); err != nil {
		return nil, fmt.Errorf("could not unmashal meta file json: %w", err)
	}

	blocksFolder := path.Join(matrixFolder, "special_blocks")

	filePath := path.Join(blocksFolder, "world_blocks.csv")
	worldBlocks, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	wb := worldBlocks[0][len(worldBlocks[0])-1]

	filePath = path.Join(blocksFolder, "sector_blocks.csv")
	sectorBlocks, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	sbs := map[string]string{}
	for _, row := range sectorBlocks {
		sbs[row[0]] = row[len(row)-1]
	}

	filePath = path.Join(blocksFolder, "arena_blocks.csv")
	arenaBlocks, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	abs := map[string]string{}
	for _, row := range arenaBlocks {
		abs[row[0]] = row[len(row)-1]
	}

	filePath = path.Join(blocksFolder, "game_object_blocks.csv")
	objectBlocks, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	obs := map[string]string{}
	for _, row := range objectBlocks {
		obs[row[0]] = row[len(row)-1]
	}

	filePath = path.Join(blocksFolder, "game_object_blocks.csv")
	spawningLocations, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	sls := map[string]string{}
	for _, row := range spawningLocations {
		sls[row[0]] = row[len(row)-1]
	}

	mazeFolder := path.Join(matrixFolder, "maze")

	filePath = path.Join(mazeFolder, "collision_maze.csv")
	cm, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	filePath = path.Join(mazeFolder, "sector_maze.csv")
	sm, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	filePath = path.Join(mazeFolder, "arena_maze.csv")
	am, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	filePath = path.Join(mazeFolder, "game_object_maze.csv")
	gom, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}
	filePath = path.Join(mazeFolder, "spawning_location_maze.csv")
	slm, err := readCSVFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read csv file %s: %w", filePath, err)
	}

	collisionMaze := [][]bool{}
	sectorMaze := [][]string{}
	arenaMaze := [][]string{}
	gameObjectMaze := [][]string{}
	spawningLocationMaze := [][]string{}
	for i := 0; i < len(cm[0]); i += meta.MazeWidth {
		cmr := make([]bool, 0, meta.MazeWidth)
		for _, v := range cm[0][i : i+meta.MazeWidth] {
			cmr = append(cmr, v != "0")
		}
		collisionMaze = append(collisionMaze, cmr)
		sectorMaze = append(sectorMaze, sm[0][i:i+meta.MazeWidth])
		arenaMaze = append(arenaMaze, am[0][i:i+meta.MazeWidth])
		gameObjectMaze = append(gameObjectMaze, gom[0][i:i+meta.MazeWidth])
		spawningLocationMaze = append(spawningLocationMaze, slm[0][i:i+meta.MazeWidth])
	}

	tiles := [][]maze.Tile{}
	for i := 0; i < meta.MazeHeight; i += 1 {
		row := make([]maze.Tile, 0, meta.MazeWidth)
		for j := 0; j < meta.MazeWidth; j += 1 {
			var tile maze.Tile
			tile.Path = memory.ParsePath(wb)
			if t, ok := sbs[sectorMaze[i][j]]; ok {
				tile.Path = tile.Path.Copy(memory.PathWithSector(t))
			}
			if t, ok := abs[arenaMaze[i][j]]; ok {
				tile.Path = tile.Path.Copy(memory.PathWithArena(t))
			}
			if t, ok := obs[gameObjectMaze[i][j]]; ok {
				tile.Path = tile.Path.Copy(memory.PathWithObject(t))
			}
			if t, ok := sls[spawningLocationMaze[i][j]]; ok {
				tile.SpawningLocation = t
			}
			if t := collisionMaze[i][j]; t {
				tile.Collision = t
			}

			tile.Events = map[maze.Event]struct{}{}
			if tile.Path.IsObject() {
				tile.Events[maze.Event{SPO: memory.SPO{
					Subject:   tile.Path.ToString(),
					Predicate: "",
					Object:    "",
				}, Description: ""}] = struct{}{}
			}

			row = append(row, tile)
		}

		tiles = append(tiles, row)
	}

	addressTiles := map[memory.Path][]maze.TilePos{}
	for i := 0; i < meta.MazeHeight; i += 1 {
		for j := 0; j < meta.MazeHeight; j += 1 {
			addresses := []memory.Path{}
			tile := tiles[i][j]
			tileLevel := tile.Path.Level()
			if tileLevel >= memory.PathLevelSector {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelSector))
			}
			if tileLevel >= memory.PathLevelArena {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelArena))
			}
			if tileLevel >= memory.PathLevelObject {
				addresses = append(addresses, tile.Path.AtLevel(memory.PathLevelObject))
			}
			if tile.SpawningLocation != "" {
				addresses = append(addresses, memory.SpecialPath(memory.PathStateSpawningLocation, tile.SpawningLocation))
			}

			for _, add := range addresses {
				addressTiles[add] = append(addressTiles[add], maze.TilePos{X: j, Y: i})
			}
		}
	}

	return maze.New(meta.WorldName, mazeName, meta.MazeWidth, meta.MazeHeight, meta.SquareTileSize, collisionMaze, tiles), nil
}

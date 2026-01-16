package simulationloader

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/agent"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
	"github.com/fvdveen/generative_agents/simulation_server/server"
)

type MovementPersona struct {
	Movement    Position    `json:"movement"`
	Pronunciato string      `json:"pronunciato"`
	Description string      `json:"description"`
	Chat        []Utterance `json:"chat"`
}

type MovementMeta struct {
	CurrentTime CurrentTime `json:"curr_time"`
}

type Movements struct {
	Personas map[string]MovementPersona `json:"persona"`
	Meta     MovementMeta               `json:"meta"`
}

type FileStorage struct {
	SimulationsFolder string
	BackupFolder      string

	Simulation string
	Maze       string
}

func (fs FileStorage) movementFolder() string {
	return path.Join(fs.SimulationsFolder, fs.Simulation, "movement")
}

func (fs FileStorage) environmentFolder() string {
	return path.Join(fs.SimulationsFolder, fs.Simulation, "environment")
}

func (fs FileStorage) metaFolder() string {
	return path.Join(fs.SimulationsFolder, fs.Simulation, "reverie")
}

func (fs FileStorage) personaFolder(name string) string {
	return path.Join(fs.SimulationsFolder, fs.Simulation, "personas", name, "bootstrap_memory")
}

func (fs *FileStorage) backupFolder(step int) string {
	return path.Join(fs.BackupFolder, fs.Simulation, strconv.Itoa(step))
}

func (fs *FileStorage) SaveMovements(step int, personaMovements map[string]server.PersonaMovement, currTime time.Time) error {
	movements := Movements{
		Personas: map[string]MovementPersona{},
		Meta: MovementMeta{
			CurrentTime: CurrentTime(currTime),
		},
	}

	for n, m := range personaMovements {
		var chat []Utterance
		for _, utt := range m.Chat {
			chat = append(chat, Utterance{
				utt.Speaker,
				utt.Sentence,
			})
		}
		movements.Personas[n] = MovementPersona{
			Movement:    Position{X: m.Tile.X, Y: m.Tile.Y},
			Pronunciato: m.Pronunciato,
			Description: m.Event.Description,
			Chat:        chat,
		}
	}

	personas := map[string]EnvironmentPersona{}
	for n, m := range personaMovements {
		personas[n] = EnvironmentPersona{
			Maze: fs.Maze,
			X:    m.Tile.X,
			Y:    m.Tile.Y,
		}
	}

	env := Environment{
		Personas: personas,
	}

	p := path.Join(fs.movementFolder(), fmt.Sprintf("%d.json", step))
	if err := writeJson(p, movements); err != nil {
		return fmt.Errorf("Could not save movement: %w", err)
	}

	p = path.Join(fs.environmentFolder(), fmt.Sprintf("%d.json", step+1))
	if err := writeJson(p, env); err != nil {
		return fmt.Errorf("Could not write save environment: %w", err)
	}

	return nil
}

func (fs *FileStorage) SaveSimulation(srv *server.Server) error {
	names := make([]string, 0, len(srv.Personas))
	for n, p := range srv.Personas {
		names = append(names, n)

		if err := fs.SavePersona(p); err != nil {
			return fmt.Errorf("could not save persona %s: %w", n, err)
		}
	}

	meta := SimulationMeta{
		ForkSimCode:    srv.ForkedSim,
		StartDate:      StartDate(srv.StartTime),
		CurrTime:       CurrentTime(srv.CurrentTime),
		SecondsPerStep: int(srv.TimeStep / time.Second),
		MazeName:       srv.Maze.Folder(),
		PersonaNames:   names,
		Step:           srv.Step,
	}

	if err := writeJson(path.Join(fs.metaFolder(), "meta.json"), meta); err != nil {
		return fmt.Errorf("could not save meta: %w", err)
	}

	return nil
}

func (fs *FileStorage) savePersonaState(p *agent.Persona) error {
	state := p.State()

	sched := make([]Plan, 0, len(state.DailySchedule))
	origSched := make([]Plan, 0, len(state.OriginalDailySchedule))

	for _, plan := range state.DailySchedule {
		sched = append(sched, Plan{
			Activity: plan.Activity,
			Duration: plan.Duration,
		})
	}
	for _, plan := range state.OriginalDailySchedule {
		origSched = append(origSched, Plan{
			Activity: plan.Activity,
			Duration: plan.Duration,
		})
	}

	var chattingWith *string
	if state.ChattingWith != "" {
		chattingWith = &state.ChattingWith
	}

	var chat []Utterance
	for _, utt := range state.Chat {
		chat = append(chat, Utterance{
			Speaker:   utt.Speaker,
			Utterance: utt.Sentence,
		})
	}

	var chatEndTime *time.Time
	if !state.ChatEndTime.IsZero() {
		chatEndTime = &state.ChatEndTime
	}

	var plannedPath []Position
	for _, pos := range state.PlannedPath {
		plannedPath = append(plannedPath, Position{
			X: pos.X,
			Y: pos.Y,
		})
	}

	scratch := PersonaState{
		VisionR:                 state.VisionRadius,
		AttBandwidth:            state.AttentionBandwidth,
		Retention:               state.Retention,
		CurrTime:                CurrentTime(state.CurrentTime),
		CurrTile:                []int{state.Position.X, state.Position.Y},
		DailyPlanReq:            state.DailyPlanRequirements,
		Name:                    p.Name(),
		FirstName:               state.FirstName,
		LastName:                state.LastName,
		Age:                     state.Age,
		Innate:                  state.InnateTraits,
		Learned:                 state.LearnedTraits,
		Currently:               state.CurrentPlans,
		Lifestyle:               state.Lifestyle,
		LivingArea:              state.LivingArea.ToString(),
		RecencyW:                state.RecencyWeight,
		RelevanceW:              state.RelevanceWeight,
		ImportanceW:             state.ImportanceWeight,
		ValenceW:                state.ValenceWeight,
		RecencyDecay:            state.RecencyDecay,
		ImportanceTriggerMax:    state.ReflectionTrigger,
		ImportanceTriggerCurr:   state.CurrentReflectionTrigger,
		ImportanceEleN:          state.ReflectionElements,
		DailyReq:                state.DailyPlan,
		FDailySchedule:          sched,
		FDailyScheduleHourlyOrg: origSched,
		ActAddress:              state.ActivityAddress.ToString(),
		ActStartTime:            CurrentTime(state.ActivityStartTime),
		ActDuration:             int(state.ActivityDuration.Minutes()),
		ActDescription:          state.ActivityDescription,
		ActPronunciatio:         state.ActivityPronunciato,
		ActEvent: SPO{
			Subject:   state.ActivitySPO.Subject,
			Predicate: state.ActivitySPO.Predicate,
			Object:    state.ActivitySPO.Object,
		},
		ActObjDescription:  state.ActivityObjectDescription,
		ActObjPronunciatio: state.ActivityObjectPronunciato,
		ActObjEvent: SPO{
			Subject:   state.ActivityObjectSPO.Subject,
			Predicate: state.ActivityObjectSPO.Predicate,
			Object:    state.ActivityObjectSPO.Object,
		},
		ChattingWith:       chattingWith,
		Chat:               chat,
		ChattingWithBuffer: state.ChattingWithBuffer,
		ChattingEndTime:    (*CurrentTime)(chatEndTime),
		ActPathSet:         state.ActivityPathSet,
		PlannedPath:        plannedPath,
	}

	if err := writeJson(path.Join(fs.personaFolder(p.Name()), "scratch.json"), scratch); err != nil {
		return fmt.Errorf("could not save persona %s state: %w", p.Name(), err)
	}

	return nil
}

func (fs *FileStorage) saveSpatialMemory(name string, store *memory.Spatial) error {
	mem := map[string]map[string]map[string][]string{}

	for world, sectors := range store.Worlds() {
		mem[world] = make(map[string]map[string][]string)
		for sector, arenas := range sectors {
			mem[world][sector] = make(map[string][]string)
			for arena, objects := range arenas {
				mem[world][sector][arena] = make([]string, 0, len(objects))
				for obj := range objects {
					mem[world][sector][arena] = append(mem[world][sector][arena], obj)
				}
			}
		}
	}

	if err := writeJson(path.Join(fs.personaFolder(name), "spatial_memory.json"), mem); err != nil {
		return fmt.Errorf("could not save persona %s spatial memory: %w", name, err)
	}

	return nil
}

func (fs *FileStorage) saveAssociativeMemory(name string, store *memory.Associative) error {
	if err := writeJson(path.Join(fs.personaFolder(name), "associative_memory", "embeddings.json"), store.Embeddings()); err != nil {
		return fmt.Errorf("could not save persona %s associative embeddings: %w", name, err)
	}

	if err := writeJson(path.Join(fs.personaFolder(name), "associative_memory", "kw_strength.json"), KwStength{
		Thoughts: store.ThoughtKeywordStrength(),
		Events:   store.EventKeywordStrength(),
	}); err != nil {
		return fmt.Errorf("could not save persona %s associative keyword strength: %w", name, err)
	}

	nodes := map[string]MemoryNode{}
	for _, node := range store.Nodes() {
		var filling []any
		switch node.Type {
		case memory.NodeTypeChat:
			for _, utt := range node.Chat {
				filling = append(filling, Utterance{
					Speaker:   utt.Speaker,
					Utterance: utt.Sentence,
				})
			}
		case memory.NodeTypeEvent, memory.NodeTypeThought:
			for _, id := range node.Evidence {
				filling = append(filling, fmt.Sprintf("node_%d", id))
			}
		default:
			panic(fmt.Sprintf("unexpected memory.NodeType: %#v", node.Type))
		}

		nodes[fmt.Sprintf("node_%d", node.Id)] = MemoryNode{
			NodeCount:    node.NodeCount,
			TypeCount:    node.TypeCount,
			Type:         node.Type.ToString(),
			Depth:        node.Depth,
			Created:      MemoryTime(node.Created),
			Expiration:   (*MemoryTime)(node.Expiration),
			Subject:      node.Subject,
			Predicate:    node.Predicate,
			Object:       node.Object,
			Description:  node.Description,
			EmbeddingKey: node.EmbeddingKey,
			Poignancy:    node.Importance,
			Valence:      node.Valence,
			Keywords:     node.Keywords,
			Filling:      filling,
		}
	}

	if err := writeJson(path.Join(fs.personaFolder(name), "associative_memory", "nodes.json"), nodes); err != nil {
		return fmt.Errorf("could not save persona %s associative nodes: %v", err)
	}

	return nil
}

func (fs *FileStorage) SavePersona(p *agent.Persona) error {
	if err := fs.savePersonaState(p); err != nil {
		return err
	}

	assoc, spatial := p.Memory()

	if err := fs.saveSpatialMemory(p.Name(), spatial); err != nil {
		return err
	}

	if err := fs.saveAssociativeMemory(p.Name(), assoc); err != nil {
		return err
	}

	return nil
}

func writeJson(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal JSON: %w", err)
	}

	if err := writeFileWithDirs(path, data, 0o644); err != nil {
		return fmt.Errorf("could not write file to %s: %w", path, err)
	}

	return nil
}

func writeFileWithDirs(path string, data []byte, perm os.FileMode) error {
	// Ensure parent directories exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write the file
	return os.WriteFile(path, data, perm)
}

func (fs *FileStorage) Backup(step int) error {
	return copyDirFilesOnly(path.Join(fs.SimulationsFolder, fs.Simulation), fs.backupFolder(step))
}

func copyDirFilesOnly(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode())
		}

		// Regular file
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file encountered (expected only files/dirs): %s", path)
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Ensure parent dir exists (useful if dst root existed but some subdirs didn't)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

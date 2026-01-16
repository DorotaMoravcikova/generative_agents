package agent

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/llm"
	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type NewDayType int

const (
	NewDayTypeNoNewDay NewDayType = iota
	NewTypeDayFirstDay
	NewDayTypeNewDay
)

func (p *Persona) reviseIdentity() {
	/* NOTE(Friso): In the original code the currently field of the persona's state is only updated once per day, whilst they are sleeping probably.
	   This cannot be correct in any way, also they only update the daily_plan_req which gets fed into the prompts via the personas identity stable set,
	   however this can then conflict with the daily requirements as specified in the field in its state.
	*/
	focalPoints := []string{
		fmt.Sprintf("%s's plan for %s.", p.Name(), p.CurrentTime().Format("Monday January 02")),
		fmt.Sprintf("Important recent events for %s's life.", p.Name()),
	}
	retrieved := p.retrieveForFocalPoints(focalPoints)

	statements := []string{}
	for _, nodes := range retrieved {
		for _, node := range nodes {
			mem := p.GetMemory(node)

			statements = append(statements, fmt.Sprintf("%s: %s", mem.Created.Format("Monday January 02 -- 15:04 PM"), mem.EmbeddingKey))
		}
	}

	note := p.cognition.GeneratePlanningNote(p, statements)
	feelings := p.cognition.GeneratePlanningFeelings(p, statements)

	newStatus := p.cognition.GenerateCurrentPlans(p, note, feelings)

	p.state.CurrentPlans = newStatus

	dailyReq := p.cognition.GenerateNewDailyRequirements(p)
	p.state.DailyPlanRequirements = dailyReq
}

func (p *Persona) longTermPlanning(newDay NewDayType) {
	var wakeUpHour time.Time

	switch newDay {
	case NewTypeDayFirstDay:
		wakeUpHour = p.cognition.GenerateWakeUpHour(p)
		p.state.DailyPlan = p.cognition.GenerateDailyPlan(p, wakeUpHour)
	case NewDayTypeNewDay:
		/* NOTE(Friso): In the current version of the code of the original paper
		   that is github the agents do not create a new daily plan every morning.
		   To me it seems like that is a mistake, they even say that it should happen.
		   Thus that happens here.
		*/
		p.reviseIdentity()

		wakeUpHour = p.cognition.GenerateWakeUpHour(p)
		p.state.DailyPlan = p.cognition.GenerateDailyPlan(p, wakeUpHour)

		// NOTE(Friso): In the original code they state that a new daily plan _should_ be created here, but it isn't
	default:
		panic("this should be unreachable")
	}

	p.state.DailySchedule = p.cognition.GenerateHourlySchedule(p, wakeUpHour)
	p.state.OriginalDailySchedule = slices.Clone(p.state.DailySchedule)

	thought := fmt.Sprintf(
		"This is %s's plan for %s: %s.",
		p.name,
		p.state.CurrentTime.Format("Monday January 02"),
		strings.Join(p.state.DailyPlan, ", "))
	createdAt := p.state.CurrentTime
	expiratesAt := createdAt.Add(30 * 24 * time.Hour)
	spo := memory.SPO{
		Subject:   p.name,
		Predicate: "plan",
		Object:    p.state.CurrentTime.Format("Monday January 02"),
	}
	keywords := []string{"plan"}
	// NOTE(Friso): I feel like we should probably get the agents to generate this value for themselves,
	// a daily planning of them studying all day should be less important than them going an a date for example.
	importance := 5
	valence := 0
	embedding := p.GetEmbedding(thought)
	p.addThoughtToMemory(spo, thought, keywords, importance, valence, make([]memory.NodeId, 0), createdAt, &expiratesAt, thought, embedding)
}

func (p *Persona) determineActivity(maze *maze.Maze) {
	shouldDecomposeActivity := func(desc string, dur int) bool {
		if !strings.Contains(desc, "sleep") && !strings.Contains("bed", desc) {
			return true
		} else if strings.Contains(desc, "sleeping") || strings.Contains(desc, "asleep") || strings.Contains(desc, "in bed") {
			return false
		} else if strings.Contains(desc, "sleep") || strings.Contains(desc, "bed") {
			if dur > 60 {
				return false
			}
		}

		return true
	}

	currIndex := p.state.GetDailyPlanIndex()
	currIndexInHour := p.state.GetDailyPlanIndexInMinutes(60)

	if currIndex == 0 {
		plan := p.state.DailySchedule[currIndex]
		if plan.Duration >= 60 && shouldDecomposeActivity(plan.Activity, plan.Duration) {
			decomposedPlan := p.cognition.GeneratePlanDecomposition(p, plan)
			before, after := slices.Clone(p.state.DailySchedule[:currIndex]), p.state.DailySchedule[currIndex+1:]
			p.state.DailySchedule = append(
				before,
				decomposedPlan...,
			)
			p.state.DailySchedule = append(
				p.state.DailySchedule,
				after...,
			)
		}
		if currIndexInHour+1 < len(p.state.DailySchedule) {
			plan := p.state.DailySchedule[currIndexInHour+1]
			if plan.Duration >= 60 && shouldDecomposeActivity(plan.Activity, plan.Duration) {
				decomposedPlan := p.cognition.GeneratePlanDecomposition(p, plan)
				before, after := slices.Clone(p.state.DailySchedule[:currIndexInHour+1]), p.state.DailySchedule[currIndexInHour+2:]
				p.state.DailySchedule = append(
					before,
					decomposedPlan...,
				)
				p.state.DailySchedule = append(
					p.state.DailySchedule,
					after...,
				)
			}
		}
	}

	if currIndexInHour < len(p.state.DailySchedule) {
		// NOTE(Friso): In the original code they don't decompose activitys after 11 pm.
		// I'm not sure exactly why they do this.
		if p.state.CurrentTime.Hour() < 23 {
			plan := p.state.DailySchedule[currIndexInHour]
			if plan.Duration >= 60 && shouldDecomposeActivity(plan.Activity, plan.Duration) {
				decomposedPlan := p.cognition.GeneratePlanDecomposition(p, plan)
				before, after := slices.Clone(p.state.DailySchedule[:currIndexInHour]), p.state.DailySchedule[currIndexInHour+1:]
				p.state.DailySchedule = append(
					before,
					decomposedPlan...,
				)
				p.state.DailySchedule = append(
					p.state.DailySchedule,
					after...,
				)
			}
		}
	}

	const dayDuration = 24 * time.Hour
	scheduledDuration := time.Duration(0)
	for _, plan := range p.state.DailySchedule {
		scheduledDuration += time.Duration(plan.Duration) * time.Minute
	}

	if scheduledDuration < dayDuration {
		p.state.DailySchedule = append(p.state.DailySchedule,
			llm.Plan{
				Activity: "sleeping",
				Duration: int(dayDuration.Minutes()) - int(scheduledDuration.Minutes()),
			})
	} else if scheduledDuration > dayDuration {
		panic("TODO: handle daily plan longer than day")
	}

	currPlan := p.state.DailySchedule[currIndex]

	world := maze.GetTile(p.state.Position).Path.Get(memory.PathLevelWorld)
	sector := p.cognition.GenerateActivitySector(p, maze, currPlan.Activity, world)
	arena := p.cognition.GenerateActivityArena(p, maze, currPlan.Activity, world, sector)
	activityAddress := memory.NewPath(
		memory.PathWithWorld(world),
		memory.PathWithSector(sector),
		memory.PathWithArena(arena),
	)
	activityObject := p.cognition.GenerateActivityObject(p, maze, currPlan.Activity, activityAddress)
	activityAddress = activityAddress.Copy(memory.PathWithObject(activityObject))

	activityPronunciato := p.cognition.GenerateActivityPronunciato(p, currPlan.Activity)
	activitySPO := p.cognition.GenerateActivitySPO(p, currPlan.Activity)

	// Since the persona's activitys also influence object states we need to set those up
	activityObjectDescription := p.cognition.GenerateActivityObjectDescription(p, activityObject, currPlan.Activity)
	activityObjectPronunciato := p.cognition.GenerateActivityObjectPronunciato(p, activityObjectDescription)
	activityObjectSPO := p.cognition.GenerateActivityObjectSPO(p, activityObject, activityObjectDescription)

	// NOTE(Friso): In the original code they state that adding a new activity means adding it to some kind of activity queue,
	// this is not what happens, they just set the current activity, so that is the behaviour I'll copy
	p.state.SetActivity(
		p.ctx.Log,
		activityAddress,
		time.Duration(currPlan.Duration)*time.Minute,
		currPlan.Activity,
		activityPronunciato,
		activitySPO,
		activityObjectDescription,
		activityObjectPronunciato,
		activityObjectSPO)
}

func (p *Persona) chooseRetrieved(retrieved map[string]relevantNodes) (relevantNodes, bool) {
	// NOTE(Friso): We delete from retrieved here, mutating it, be careful as this might cause unexpected behaviour when code is changed
	// In the papers code they remove self events so we do too
	maps.DeleteFunc(retrieved, func(decs string, ev relevantNodes) bool {
		return p.associativeMemory.GetNode(ev.currEvent).Subject == p.name
	})

	priority := make([]relevantNodes, 0)
	// From what I understand from the original code reacting to persona events takes prescedent over reacting to object events
	for _, rn := range retrieved {
		node := p.associativeMemory.GetNode(rn.currEvent)
		if !strings.Contains(node.Subject, ":") && node.Subject != p.name {
			priority = append(priority, rn)
		}
	}
	if len(priority) > 0 {
		return priority[rand.Intn(len(priority))], true
	}

	for desc, rn := range retrieved {
		if !strings.Contains(desc, "is idle") {
			priority = append(priority, rn)
		}
	}
	if len(priority) > 0 {
		return priority[rand.Intn(len(priority))], true
	}

	return relevantNodes{}, false
}

func letsTalk(init, target *Persona, focussed relevantNodes) bool {
	if init.state.ActivityAddress.IsEmpty() ||
		init.state.ActivityDescription == "" ||
		target.state.ActivityAddress.IsEmpty() ||
		target.state.ActivityDescription == "" {
		return false
	}

	if strings.Contains(init.state.ActivityDescription, "sleeping") ||
		strings.Contains(target.state.ActivityDescription, "sleeping") {
		return false
	}

	// NOTE(Friso): I'm not sure why this case is here but they have it in the original code
	if init.state.CurrentTime.Hour() == 23 {
		return false
	}

	if target.state.ActivityAddress.HasState(memory.PathStateWaiting) {
		return false
	}

	if init.state.ChattingWith != "" || target.state.ChattingWith != "" {
		return false
	}

	if p, ok := init.state.ChattingWithBuffer[target.name]; ok && p > 0 {
		return false
	}

	events := make([]memory.NodeId, 0, len(focussed.events))
	thoughts := make([]memory.NodeId, 0, len(focussed.thoughts))
	for node := range focussed.events {
		events = append(events, node)
	}
	for node := range focussed.thoughts {
		thoughts = append(thoughts, node)
	}

	return init.cognition.GenerateDecideToTalk(init, target, events, thoughts)
}

// The name is copied from the orignal code but its deceptive, this function actually decides whether init should wait on target to finish their activity.
func letsReact(init, target *Persona, focussed relevantNodes) (mode string, ok bool) {
	if init.state.ActivityAddress.IsEmpty() ||
		init.state.ActivityDescription == "" ||
		target.state.ActivityAddress.IsEmpty() ||
		target.state.ActivityDescription == "" {
		return "", false
	}

	if strings.Contains(init.state.ActivityDescription, "sleeping") ||
		strings.Contains(target.state.ActivityDescription, "sleeping") {
		return "", false
	}

	// NOTE(Friso): I'm not sure why this case is here but they have it in the original code
	if init.state.CurrentTime.Hour() == 23 {
		return "", false
	}

	if strings.Contains(target.state.ActivityDescription, "waiting") {
		return "", false
	}

	if len(init.state.PlannedPath) == 0 {
		return "", false
	}
	// NOTE(Friso): I don't fully understand why skip reacting if targets activities have different addresses,
	// to me it seems like this would prevent personas from reacting to each other even if they can see each other,
	// but its in the original code; oh well.

	// If the address of the init and target personas are different it means that they are going to (?) different game zones,
	// so they should not interact.
	if init.state.ActivityAddress != target.state.ActivityAddress {
		return "", false
	}

	events := make([]memory.NodeId, 0, len(focussed.events))
	thoughts := make([]memory.NodeId, 0, len(focussed.thoughts))
	for node := range focussed.events {
		events = append(events, node)
	}
	for node := range focussed.thoughts {
		thoughts = append(thoughts, node)
	}

	shouldWait := init.cognition.GenerateDecideToWait(init, target, events, thoughts)
	if shouldWait {
		return fmt.Sprintf("wait: %s",
				target.state.ActivityStartTime.
					Add(target.state.ActivityDuration).
					Format("January 02, 2006, 15:04:05")),
			true
	}

	return "", false
}

func (p *Persona) shouldReact(focussedEvent relevantNodes, personas map[string]*Persona) (mode string, ok bool) {
	if p.state.ChattingWith != "" {
		return "", false
	} else if p.state.ActivityAddress.HasState(memory.PathStateWaiting) {
		return "", false
	}

	currEvent := p.associativeMemory.GetNode(focussedEvent.currEvent)

	if !strings.Contains(currEvent.Subject, ":") {
		target, ok := personas[currEvent.Subject]
		if !ok || p.name == target.name {
			// Target does not exist or we are reaction to ourselves
			return "", false
		}

		if letsTalk(p, target, focussedEvent) {
			return fmt.Sprintf("chat with %s", currEvent.Subject), true
		}

		return letsReact(p, personas[currEvent.Subject], focussedEvent)
	}

	return "", false
}

func (p *Persona) createReact(summary string, duration int, address memory.Path, spo memory.SPO, actStartTime time.Time, pronunciato string, chattingWith string, chat []memory.Utterance, chattingWithBuffer map[string]int, chatEndTime time.Time) {
	minSum := 0
	for i := 0; i < p.state.GetOriginalDailyPlanIndex(); i += 1 {
		minSum += p.state.OriginalDailySchedule[i].Duration
	}
	startTime := p.StartOfDay().Add(time.Duration(minSum) * time.Minute)

	var endTime time.Time
	if d := p.state.OriginalDailySchedule[p.state.GetOriginalDailyPlanIndex()].Duration; d >= 120 {
		endTime = startTime.Add(time.Duration(d) * time.Minute)
	} else if len(p.state.OriginalDailySchedule) > p.state.GetOriginalDailyPlanIndex()+1 {
		d1 := p.state.OriginalDailySchedule[p.state.GetOriginalDailyPlanIndex()].Duration
		d2 := p.state.OriginalDailySchedule[p.state.GetOriginalDailyPlanIndex()+1].Duration

		endTime = startTime.Add(time.Duration(d1+d2) * time.Minute)
	} else {
		endTime = startTime.Add(2 * time.Hour)
	}

	durSum := p.StartOfDay()
	startIndex := -1
	endIndex := -1
	for i, plan := range p.state.DailySchedule {
		if !durSum.Before(startTime) && startIndex == -1 {
			startIndex = i
		}
		if !durSum.Before(endTime) && endIndex == -1 {
			endIndex = i
		}
		durSum = durSum.Add(time.Duration(plan.Duration) * time.Minute)
	}

	newPlans := p.cognition.GenerateReactionScheduleUpdate(p, llm.Plan{Duration: duration, Activity: summary}, startTime, endTime)

	before, after := slices.Clone(p.state.DailySchedule[:startIndex]), p.state.DailySchedule[endIndex:]
	p.state.DailySchedule = append(
		before,
		newPlans...,
	)
	p.state.DailySchedule = append(
		p.state.DailySchedule,
		after...,
	)

	dur := time.Duration(duration) * time.Minute
	if chattingWith != "" {
		p.state.SetChatActivity(p.ctx.Log, address, dur, summary, pronunciato, spo, chattingWith, chat, chattingWithBuffer, chatEndTime)
	} else {
		p.state.SetActivity(p.ctx.Log, address, dur, summary, pronunciato, spo, "", "", memory.SPO{})
	}
}

func getLastN[T any](elems []T, n int) []T {
	if len(elems) < n {
		return elems
	}

	return elems[len(elems)-n:]
}

func (p *Persona) iterativeGenerateConversation(target *Persona, maze *maze.Maze) (chat []memory.Utterance, duration int) {
	generateUtterance := func(init, target *Persona, chat []memory.Utterance) (memory.Utterance, bool) {
		relationshipMemories := p.retrieveForFocalPoints([]string{target.name}, withRetrievalCount(50))
		nodes := []memory.NodeId{}
		for _, ns := range relationshipMemories {
			nodes = append(nodes, ns...)
		}
		relationship := p.cognition.GenerateRelationshipSummary(p, target, nodes)

		focalPoints := []string{relationship, fmt.Sprintf("%s is %s", target.name, target.ActivityDescription())}
		lastUtt := getLastN(chat, 4)
		for _, utt := range lastUtt {
			focalPoints = append(focalPoints, "%s: %s\n", utt.Speaker, utt.Sentence)
		}

		retrieved := init.retrieveForFocalPoints(focalPoints, withRetrievalCount(15))
		nodes = []memory.NodeId{}
		for _, ns := range retrieved {
			nodes = append(nodes, ns...)
		}

		return init.cognition.GenerateOneUtterance(init, target, maze, chat, nodes, relationship)
	}

	length := 0
	for i := 0; i < 8; i += 1 {
		utt, done := generateUtterance(p, target, chat)
		chat = append(chat, utt)
		if done {
			break
		}

		utt, done = generateUtterance(target, p, chat)
		chat = append(chat, utt)
		if done {
			break
		}

		length += len(utt.Speaker) + len(utt.Sentence) + 3
	}

	return chat, int(float64(length)/8) / 30
}

func (p *Persona) chatReact(maze *maze.Maze, reactionMode string, personas map[string]*Persona) {
	target := personas[strings.TrimPrefix(reactionMode, "chat with ")]

	conversation, duration := p.iterativeGenerateConversation(target, maze)
	summary := p.cognition.GenerateConversationSummary(p, conversation)

	endOfMinute := p.state.CurrentTime
	if endOfMinute.Second() != 0 {
		endOfMinute = endOfMinute.Add(time.Duration(endOfMinute.Second()) * time.Second)
	}
	chatEndTime := endOfMinute.Add(time.Duration(duration) * time.Minute)

	react := func(p, other *Persona) {
		address := memory.SpecialPath(memory.PathStatePersona, other.name)
		spo := memory.SPO{
			Subject:   p.name,
			Predicate: "chat with",
			Object:    other.name,
		}

		chattingWith := map[string]int{other.name: 800}
		pronunciato := "💬"

		p.createReact(summary, duration, address, spo, p.state.ActivityStartTime, pronunciato, other.name, conversation, chattingWith, chatEndTime)
	}

	react(p, target)
	react(target, p)
}

func (p *Persona) waitReact(reactionMode string) {
	// NOTE(Friso): Because of this it is important that descriptions do not contain parentheses by themselves, only we should insert them
	// its kind of a dumb design descition but oh well.
	descStart := strings.Index(p.state.ActivityDescription, "(")
	descEnd := strings.Index(p.state.ActivityDescription, ")")
	desc := p.state.ActivityDescription

	if descStart != -1 && descEnd != -1 {
		desc = p.state.ActivityDescription[descStart+1 : descEnd]
	}

	insertedActivity := fmt.Sprintf("waiting to start %s", desc)
	endTime, err := time.Parse("January 02, 2006, 15:04:05", strings.TrimPrefix(reactionMode, "wait: "))
	if err != nil {
		panic(fmt.Errorf("unable to parse formatted time: %w", err))
	}
	activityDuration := int(endTime.Sub(p.state.CurrentTime).Minutes()) + 1

	address := memory.SpecialPath(memory.PathStateWaiting, fmt.Sprintf(memory.WaitingArgFormat, p.state.Position.X, p.state.Position.Y))
	spo := memory.SPO{
		Subject:   p.name,
		Predicate: "waiting to start",
		Object:    desc,
	}

	pronunciatio := "⌛"

	p.createReact(insertedActivity, activityDuration, address, spo, time.Time{}, pronunciatio, "", []memory.Utterance{}, map[string]int{}, time.Time{})
}

func (p *Persona) plan(maze *maze.Maze, personas map[string]*Persona, retrieved map[string]relevantNodes, newDay NewDayType) memory.Path {
	// On the start of a new day the personas schedule is empty, thus we need to fill it
	if newDay != NewDayTypeNoNewDay {
		p.longTermPlanning(newDay)
	}

	if p.state.IsActivityFinished() {
		p.determineActivity(maze)
	}

	var focussedEvent relevantNodes
	var ok bool = false
	if len(retrieved) > 0 {
		focussedEvent, ok = p.chooseRetrieved(retrieved)
	}

	if ok {
		if reactionMode, ok := p.shouldReact(focussedEvent, personas); ok {
			if strings.HasPrefix(reactionMode, "chat with") {
				p.chatReact(maze, reactionMode, personas)
			} else if strings.HasPrefix(reactionMode, "wait") {
				p.waitReact(reactionMode)
			}
		}
	}

	// Clean up chat related persona state if we're not actively in a chat
	if p.state.ActivitySPO.Predicate != "chat with" {
		p.state.ChattingWith = ""
		p.state.Chat = []memory.Utterance{}
		p.state.ChatEndTime = time.Time{}
	}

	// To ensure that personas do not devolve into infinite loops of chatting with each other
	// we have a cooldown in place preventing personas from chatting again for a short time after they've chatted before.
	for name := range p.state.ChattingWithBuffer {
		if name == p.name {
			continue
		}
		p.state.ChattingWithBuffer[name] -= 1
	}

	return p.state.ActivityAddress
}

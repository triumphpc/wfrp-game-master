// Package game provides character creation workflow for WFRP 4E
package game

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CharacterCreationState represents the current step in character creation
type CharacterCreationState int

const (
	CC_Idle CharacterCreationState = iota
	CC_Name
	CC_Race
	CC_Career
	CC_Stats
	CC_Skills
	CC_Talents
	CC_Gear
	CC_Appearance
	CC_Personality
	CC_Review
	CC_Save
	CC_Complete
)

// RussianStatsMapping maps English stat codes to Russian
var RussianStatsMapping = map[string]string{
	"WS":  "ББ",
	"BS":  "ДБ",
	"S":   "СС",
	"I":   "И",
	"Ag":  "Л",
	"WP":  "О",
	"Fel": "СТ",
	"T":   "К",
}

// RussianStatsFullNames maps Russian stat codes to full names
var RussianStatsFullNames = map[string]string{
	"ББ": "Боевая Пригодность",
	"ДБ": "Дистанция Боя",
	"СС": "Сила",
	"И":  "Инициатива",
	"Л":  "Ловкость",
	"О":  "Общение",
	"СТ": "Стойкость",
	"К":  "Классовая",
}

// IsLLMQuestion detects if user input is a question for LLM
func IsLLMQuestion(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	questionPatterns := []string{"?", "как", "что такое", "объясни", "расскажи", "подробней", "помоги", "сможешь", "можешь", "расскажи", "?"}
	for _, pattern := range questionPatterns {
		if strings.Contains(input, pattern) {
			log.Printf("[LLM] Detected question pattern: %s in input: %s", pattern, input)
			return true
		}
	}
	log.Printf("[LLM] No question pattern detected in: %s", input)
	return false
}

// GetRussianStat returns Russian stat code for English stat
func GetRussianStat(english string) string {
	if russian, ok := RussianStatsMapping[english]; ok {
		return russian
	}
	return english
}

// GetRussianStatsMap converts English stats to Russian
func GetRussianStatsMap(stats map[string]int) map[string]int {
	result := make(map[string]int)
	for eng, val := range stats {
		rus := GetRussianStat(eng)
		result[rus] = val
	}
	return result
}

// CharacterCreationData holds all data during character creation
type CharacterCreationData struct {
	Name        string
	Race        string
	RaceBonusXP int
	Class       string
	Career      string
	CareerRank  string
	Status      string
	StatusLevel int
	CareerXP    int

	// Characteristics
	WS  int // Weapon Skill
	BS  int // Ballistic Skill
	S   int // Strength
	T   int // Toughness
	I   int // Initiative
	Ag  int // Agility
	Dex int // Dexterity
	Int int // Intelligence
	WP  int // Willpower
	Fel int // Fellowship

	// Secondary characteristics
	HP         int
	Fate       int
	Fortune    int
	Resilience int
	Resolve    int
	Movement   int

	// Skills from race and career
	Skills map[string]int // skillName -> rating

	// Talents from race and career
	Talents []string

	// Gear
	Gear map[string]string // item -> source

	// Money
	Money int

	// Appearance
	Age       int
	Height    string
	HairColor string
	EyeColor  string
	Features  string

	// Personality
	Strengths  []string
	Weaknesses []string
	Background string
	Motivation string

	// XP tracking
	TotalXP      int
	XPFromRace   int
	XPFromStats  int
	XPFromCareer int
	XPSpent      int

	// Creation options (how they chose)
	StatsMethod  string // "random_no_swap", "random_swap", "manual"
	CareerMethod string // "first_roll", "three_rolls", "manual"
	RaceMethod   string // "manual", "random"

	// File path for history
	BasePath string
}

// CharacterCreator manages the character creation state machine
type CharacterCreator struct {
	State CharacterCreationState
	Data  *CharacterCreationData

	// Current step input (for validation)
	currentInput string

	// LLM provider for questions
	LLMProvider interface {
		GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error)
	}

	// File path for history
	BasePath string
}

// NewCharacterCreator creates a new character creator instance
func NewCharacterCreator(basePath string) *CharacterCreator {
	return &CharacterCreator{
		State: CC_Name,
		Data: &CharacterCreationData{
			Skills:   make(map[string]int),
			Talents:  []string{},
			Gear:     make(map[string]string),
			BasePath: basePath,
		},
	}
}

// SetLLMProvider sets the LLM provider for character creation
func (cc *CharacterCreator) SetLLMProvider(provider interface {
	GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error)
}) {
	cc.LLMProvider = provider
}

// WFRPPromptForState returns a prompt explaining current step in Russian
func (cc *CharacterCreator) WFRPPromptForState() string {
	switch cc.State {
	case CC_Race:
		return "Объясни, как выбрать расу в WFRP 4E. Какие расы доступны и какие дают бонусы?"
	case CC_Career:
		return "Объясни, как выбрать карьеру в WFRP 4E. Что такое классы карьер и как они влияют на персонажа?"
	case CC_Stats:
		return "Объясни систему характеристик WFRP 4E: Боевая Пригодность (ББ), Дистанция Боя (ДБ), Сила (СС), Инициатива (И), Ловкость (Л), Общение (О), Стойкость (СТ), Классовая (К). Как они влияют на персонажа и как распределять очки?"
	case CC_Skills:
		return "Объясни систему навыков в WFRP 4E. Как выбираются навыки от расы и карьеры?"
	case CC_Talents:
		return "Объясни систему талантов в WFRP 4E. Как получаются таланты?"
	case CC_Gear:
		return "Объясни систему снаряжения в WFRP 4E. Как выбирается начальное снаряжение?"
	case CC_Appearance:
		return "Объясни, как генерируется внешность персонажа в WFRP 4E (возраст, рост, волосы, глаза)."
	default:
		return "Расскажи подробнее о создании персонажа в WFRP 4E."
	}
}

// AskLLM sends a question to LLM and returns the answer
func (cc *CharacterCreator) AskLLM(question string) (string, error) {
	if cc.LLMProvider == nil {
		return "Извини, LLM сейчас недоступен. Попробуй задать вопрос позже.", nil
	}

	prompt := fmt.Sprintf(`Ты Game Master в Warhammer Fantasy Roleplay 4th Edition.
Отвечай на вопрос игрока о правилах создания персонажа.
Ответь кратко и по существу на русском языке.

Вопрос: %s

Ответ:`, question)

	log.Printf("[LLM] Question: %s", question)

	ctx := context.Background()
	answer, err := cc.LLMProvider.GenerateRequest(ctx, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка LLM: %v", err)
	}

	log.Printf("[LLM] Raw answer: %s", answer)

	// Clean markdown from answer - replace ** with * for Telegram
	answer = strings.ReplaceAll(answer, "**", "*")
	// Escape other special chars that might break Telegram
	answer = strings.ReplaceAll(answer, "_", " ")

	log.Printf("[LLM] Cleaned answer: %s", answer)

	return answer, nil
}

// GetPrompt returns the prompt for the current state
func (cc *CharacterCreator) GetPrompt() string {
	switch cc.State {
	case CC_Name:
		return `Как тебя зовут, герой? Напиши имя персонажа.

💡 Подсказки:
• Просто напиши имя (например: Иван, Мария)
• Напиши "сгенери имя" или "сгенери сам" - я придумаю имя сам`

	case CC_Race:
		return `Выбери расу:
1. Человек (+0 XP)
2. Полурослик (+0 XP)
3. Гном (+0 XP)
4. Высший эльф (+0 XP)
5. Лесной эльф (+0 XP)

Или напиши "бросить" - случайный выбор (d100) +20 XP`

	case CC_Career:
		return `Выбери способ выбора карьеры:
1. Первый бросок принять (+50 XP)
2. Три броска - выбрать одну (+25 XP)
3. Выбрать самому (+0 XP)

Напиши номер варианта.`

	case CC_Stats:
		return `Выбери способ генерации характеристик:
1. Случайные без перестановок (+50 XP)
2. Случайные с перестановкой (+25 XP)
3. Ручное распределение 100 пунктов (0 XP)

Напиши номер варианта.
Примечание: минимум 4, максимум 18 на характеристику.`

	case CC_Skills:
		return `Теперь выберим навыки.

От расы ты получаешь:
- 3 навыка с +5 шагами развития
- 3 навыка с +3 шагами развития

От карьеры получаешь 40 шагов развития (распределить между 8 навыками).

Напиши "далее" когда будешь готов к следующему шагу.`

	case CC_Talents:
		return `Выбери таланты.

От расы и карьеры ты получаешь таланты (перечислены в правилах).

Напиши "далее" для продолжения.`

	case CC_Gear:
		return `Снаряжение.

От класса: базовые предметы (кинжал, кошелёк, одежда, еда на 1 день)
От карьеры: все предметы из строчки "Имущество" первой ступени
Деньги: рассчитываются по статусу

Напиши "далее" для продолжения.`

	case CC_Appearance:
		return `Определим внешность.

Используй 2d10 (НЕ 1d100!):
- Волосы: бросок по таблице волос твоей расы
- Глаза: бросок по таблице глаз
- Рост: формула зависит от расы
- Возраст: минимальный возраст расы + 2d10

Напиши "далее" для броска или опиши внешность сам.`

	case CC_Personality:
		return `Оживим персонажа!

Напиши:
1. Две-три сильные стороны характера (через запятую)
2. Две-три слабые стороны (через запятую)
3. Кратко: Откуда персонаж и чем занимался до этого?`

	case CC_Review:
		return cc.generateReview()

	case CC_Save:
		return "Проверь персонажа выше. Напиши 'да' для сохранения или 'нет' для отмены."

	default:
		return "Что-то пошло не так. Напиши /newchar для начала заново."
	}
}

// generateName generates a character name using LLM
func (cc *CharacterCreator) generateName() (string, bool) {
	log.Printf("[LLM] generateName called, LLMProvider: %v", cc.LLMProvider)
	if cc.LLMProvider == nil {
		log.Printf("[LLM] LLMProvider is nil!")
		return "Извини, LLM сейчас недоступен. Напиши имя персонажа вручную.", false
	}

	prompt := `Сгенерируй одно имя персонажа для Warhammer Fantasy Roleplay (человек, средневековый сеттинг Империи).
Верни только имя, без пояснений, без кавычек, без форматирования, без звездочек.`

	log.Printf("[LLM] Requesting name generation")

	ctx := context.Background()
	name, err := cc.LLMProvider.GenerateRequest(ctx, prompt, nil)
	if err != nil {
		log.Printf("[LLM] Error from provider: %v", err)
		// Return simple message without formatting
		return "Извини, не получилось сгенерировать имя. API LLM недоступен. Напиши имя вручную.", false
	}

	log.Printf("[LLM] Raw name: [%s]", name)

	// Clean up the name - remove all markdown formatting
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "**", "")
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.Trim(name, "\"«»-_")
	cc.Data.Name = name

	log.Printf("[LLM] Cleaned name: [%s]", name)

	result := fmt.Sprintf("Сгенерировано имя: %s\n\nЭто имя подходит? Напиши 'да' чтобы принять или другое имя.", name)
	log.Printf("[LLM] Result message: [%s]", result)
	return result, true
}

// processLLMQuestion handles questions to LLM
func (cc *CharacterCreator) processLLMQuestion(question string) (string, bool) {
	answer, err := cc.AskLLM(question)
	if err != nil {
		return fmt.Sprintf("Извини, произошла ошибка при запросе к LLM: %v\n\nПопробуй ещё раз или спроси по-другому.", err), false
	}

	// Add context about current step
	prompt := cc.WFRPPromptForState()

	return fmt.Sprintf("📚 *Пояснение:*\n\n%s\n\n---\n\n💡 *К текущему шагу:*\n\n%s\n\nНапиши свой ответ или задай ещё вопрос.", answer, prompt), true
}

// ProcessInput handles user input for the current state
func (cc *CharacterCreator) ProcessInput(input string) (string, bool) {
	cc.currentInput = input
	cc.saveStep()

	// Check for "generate name" command (US3)
	lowerInput := strings.ToLower(strings.TrimSpace(input))
	if cc.State == CC_Name && (lowerInput == "сгенери имя" || lowerInput == "сгенери сам" || lowerInput == "生成 имя" || lowerInput == "generate name" || strings.Contains(lowerInput, "сгенери")) {
		log.Printf("[CHAR] Detected generate name command: %s", input)
		return cc.generateName()
	}

	// Check for LLM question (US2)
	log.Printf("[CHAR] Checking if input is question: %s, state: %d", input, cc.State)
	if IsLLMQuestion(input) {
		log.Printf("[CHAR] Processing as LLM question")
		return cc.processLLMQuestion(input)
	}

	switch cc.State {
	case CC_Name:
		return cc.processName(input)

	case CC_Race:
		return cc.processRace(input)

	case CC_Career:
		return cc.processCareer(input)

	case CC_Stats:
		return cc.processStats(input)

	case CC_Skills:
		cc.State = CC_Talents
		return "Таланты:\n" + cc.getTalentsList() + "\n\nНапиши 'далее' для продолжения.", true

	case CC_Talents:
		cc.State = CC_Gear
		return "Снаряжение:\n" + cc.getGearInfo() + "\n\nНапиши 'далее' для продолжения.", true

	case CC_Gear:
		cc.State = CC_Appearance
		return cc.processAppearance("")

	case CC_Appearance:
		cc.State = CC_Personality
		return cc.GetPrompt(), true

	case CC_Personality:
		cc.processPersonality(input)
		cc.State = CC_Review
		return cc.GetPrompt(), true

	case CC_Review:
		if strings.ToLower(input) == "да" || strings.ToLower(input) == "yes" || input == "1" {
			cc.State = CC_Save
			return "Сохраняю персонажа...", true
		}
		return "Сохранение отменено. Напиши /newchar для начала заново.", false

	case CC_Save:
		return "Персонаж сохранён! Игра начинается!", false

	default:
		return cc.GetPrompt(), true
	}
}

// processName handles name input
func (cc *CharacterCreator) processName(input string) (string, bool) {
	inputLower := strings.ToLower(strings.TrimSpace(input))

	// Handle "да" to accept generated name
	if inputLower == "да" || inputLower == "yes" || inputLower == "y" {
		if cc.Data.Name != "" {
			cc.State = CC_Race
			return cc.GetPrompt(), true
		}
		return "Имя не задано. Напиши имя персонажа.", false
	}

	// Handle "давай другое" or "другое" to regenerate
	if inputLower == "давай другое" || inputLower == "другое" || inputLower == "ещё" || inputLower == "еще" || strings.Contains(inputLower, "друг") {
		if cc.LLMProvider != nil {
			return cc.generateName()
		}
		return "LLM недоступен. Напиши имя вручную.", false
	}

	// Handle "сгенери" command
	if strings.Contains(inputLower, "сгенери") || inputLower == "generate" {
		if cc.LLMProvider != nil {
			return cc.generateName()
		}
		return "LLM недоступен. Напиши имя вручную.", false
	}

	if len(input) < 2 {
		return "Имя слишком короткое. Напиши имя персонажа (минимум 2 буквы).", false
	}
	cc.Data.Name = input
	cc.State = CC_Race
	return cc.GetPrompt(), true
}

// processRace handles race selection
func (cc *CharacterCreator) processRace(input string) (string, bool) {
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for random roll
	if input == "бросить" || input == "roll" || input == "random" {
		roll := rand.Intn(100) + 1
		race := ""
		switch {
		case roll <= 90:
			race = "Человек"
			cc.Data.RaceBonusXP = 20
		case roll <= 94:
			race = "Полурослик"
			cc.Data.RaceBonusXP = 20
		case roll <= 98:
			race = "Гном"
			cc.Data.RaceBonusXP = 20
		case roll == 99:
			race = "Высший эльф"
			cc.Data.RaceBonusXP = 20
		default:
			race = "Лесной эльф"
			cc.Data.RaceBonusXP = 20
		}
		cc.Data.Race = race
		cc.Data.RaceMethod = "random"
		cc.Data.TotalXP += cc.Data.RaceBonusXP
		cc.applyRaceBonuses()
		cc.State = CC_Career
		return fmt.Sprintf("(d100 = %d) → %s!\n+20 XP (всего: %d)\n\n%s", roll, race, cc.Data.TotalXP, cc.GetPrompt()), true
	}

	// Check for number selection
	choice, err := strconv.Atoi(input)
	if err == nil {
		races := []string{"Человек", "Полурослик", "Гном", "Высший эльф", "Лесной эльф"}
		if choice >= 1 && choice <= len(races) {
			cc.Data.Race = races[choice-1]
			cc.Data.RaceMethod = "manual"
			cc.applyRaceBonuses()
			cc.State = CC_Career
			return fmt.Sprintf("Выбрал: %s\n\n%s", cc.Data.Race, cc.GetPrompt()), true
		}
	}

	// Check for race name
	races := map[string]string{
		"человек": "Человек", "1": "Человек",
		"полурослик": "Полурослик", "2": "Полурослик",
		"гном": "Гном", "3": "Гном",
		"высший эльф": "Высший эльф", "4": "Высший эльф",
		"эльф":        "Высший эльф",
		"лесной эльф": "Лесной эльф", "5": "Лесной эльф",
	}

	if race, ok := races[input]; ok {
		cc.Data.Race = race
		cc.Data.RaceMethod = "manual"
		cc.applyRaceBonuses()
		cc.State = CC_Career
		return fmt.Sprintf("Выбрал: %s\n\n%s", cc.Data.Race, cc.GetPrompt()), true
	}

	return "Не понял выбор. Напиши номер (1-5), расу или 'бросить' для случайного выбора.", false
}

// applyRaceBonuses applies racial bonuses to characteristics
func (cc *CharacterCreator) applyRaceBonuses() {
	bonuses := map[string]map[string]int{
		"Человек":     {"WS": 30, "BS": 30, "S": 20, "T": 20, "I": 30, "Ag": 30, "Dex": 30, "Int": 30, "WP": 30, "Fel": 30},
		"Полурослик":  {"WS": 20, "BS": 30, "S": 10, "T": 20, "I": 30, "Ag": 40, "Dex": 30, "Int": 30, "WP": 30, "Fel": 40},
		"Гном":        {"WS": 40, "BS": 30, "S": 30, "T": 40, "I": 20, "Ag": 20, "Dex": 30, "Int": 20, "WP": 40, "Fel": 20},
		"Высший эльф": {"WS": 40, "BS": 40, "S": 20, "T": 20, "I": 40, "Ag": 40, "Dex": 40, "Int": 40, "WP": 30, "Fel": 30},
		"Лесной эльф": {"WS": 30, "BS": 30, "S": 20, "T": 20, "I": 40, "Ag": 40, "Dex": 30, "Int": 30, "WP": 30, "Fel": 30},
	}

	if bonus, ok := bonuses[cc.Data.Race]; ok {
		cc.Data.WS = bonus["WS"]
		cc.Data.BS = bonus["BS"]
		cc.Data.S = bonus["S"]
		cc.Data.T = bonus["T"]
		cc.Data.I = bonus["I"]
		cc.Data.Ag = bonus["Ag"]
		cc.Data.Dex = bonus["Dex"]
		cc.Data.Int = bonus["Int"]
		cc.Data.WP = bonus["WP"]
		cc.Data.Fel = bonus["Fel"]
	}
}

// processCareer handles career selection
func (cc *CharacterCreator) processCareer(input string) (string, bool) {
	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)

	if err != nil {
		return "Напиши номер варианта (1-3).", false
	}

	switch choice {
	case 1:
		// First roll
		cc.Data.CareerMethod = "first_roll"
		cc.Data.CareerXP = 50
		roll := rand.Intn(100) + 1
		career := cc.getRandomCareer(roll)
		cc.Data.Career = career
		cc.Data.TotalXP += cc.Data.CareerXP

	case 2:
		// Three rolls - choose one
		cc.Data.CareerMethod = "three_rolls"
		cc.Data.CareerXP = 25

		rolls := []int{rand.Intn(100) + 1, rand.Intn(100) + 1, rand.Intn(100) + 1}
		careers := []string{cc.getRandomCareer(rolls[0]), cc.getRandomCareer(rolls[1]), cc.getRandomCareer(rolls[2])}

		msg := "Бросили три раза:\n"
		for i, c := range careers {
			msg += fmt.Sprintf("%d. %s (d100=%d)\n", i+1, c, rolls[i])
		}
		msg += "\nКакую выбираешь? Напиши номер (1-3)."

		// Store rolls for selection
		cc.Data.Career = careers[0] // temporary
		return msg, true

	case 3:
		// Manual choice - list options
		cc.Data.CareerMethod = "manual"
		cc.Data.CareerXP = 0
		cc.State = CC_Stats
		return "Выбери карьеру из списка (напиши название):\n" + cc.getCareerList() + "\n\n" + cc.GetPrompt(), true

	default:
		return "Напиши номер варианта (1-3).", false
	}

	cc.Data.TotalXP += cc.Data.CareerXP
	cc.State = CC_Stats
	return fmt.Sprintf("Карьера: %s\n+ %d XP (всего: %d)\n\n%s", cc.Data.Career, cc.Data.CareerXP, cc.Data.TotalXP, cc.GetPrompt()), true
}

// getRandomCareer returns a career based on d100 roll
func (cc *CharacterCreator) getRandomCareer(roll int) string {
	// Simplified career selection based on class
	classes := []string{"Академик", "Буржуа", "Придворный", "Крестьянин", "Рейнджер", "Ремесленник", "Учёный", "Воин"}

	// Use roll to pick class, then career
	classIdx := (roll - 1) / 12
	if classIdx >= len(classes) {
		classIdx = len(classes) - 1
	}

	class := classes[classIdx]
	careers := map[string][]string{
		"Академик":    {"Ученик", "Писарь", "Алхимик"},
		"Буржуа":      {"Торговец", "Ремесленник", "Подмастерье"},
		"Придворный":  {"Слуга", "Оруженосец", "Менестрель"},
		"Крестьянин":  {"Поденщик", "Крепостной", "Пастух"},
		"Рейнджер":    {"Охотник", "Следопыт", "Разведчик"},
		"Ремесленник": {"Кузнец", "Плотник", "Ткач"},
		"Учёный":      {"Астролог", "Целитель", "Пилот"},
		"Воин":        {"Стражник", "Наёмник", "Охранник"},
	}

	careerList := careers[class]
	career := careerList[rand.Intn(len(careerList))]

	cc.Data.Class = class
	cc.Data.Career = career
	cc.Data.CareerRank = "Ранг 1"
	cc.Data.Status = "Медный"
	cc.Data.StatusLevel = 1

	return fmt.Sprintf("%s → %s", class, career)
}

// getCareerList returns list of available careers
func (cc *CharacterCreator) getCareerList() string {
	return `
Академики: Ученик, Писарь, Алхимик
Буржуа: Торговец, Ремесленник, Подмастерье
Придворные: Слуга, Оруженосец, Менестрель
Крестьяне: Поденщик, Крепостной, Пастух
Рейнджеры: Охотник, Следопыт, Разведчик
Ремесленники: Кузнец, Плотник, Ткач
Учёные: Астролог, Целитель, Пилот
Воины: Стражник, Наёмник, Охранник
`
}

// processStats handles characteristic generation
func (cc *CharacterCreator) processStats(input string) (string, bool) {
	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)

	if err != nil {
		return "Напиши номер варианта (1-3).", false
	}

	cc.Data.StatsMethod = ""

	switch choice {
	case 1:
		// Random without swap
		cc.Data.StatsMethod = "random_no_swap"
		cc.Data.XPFromStats = 50
		cc.rollStats(false)

	case 2:
		// Random with swap
		cc.Data.StatsMethod = "random_swap"
		cc.Data.XPFromStats = 25
		cc.rollStats(true)

	case 3:
		// Manual - ask for values
		cc.Data.StatsMethod = "manual"
		cc.Data.XPFromStats = 0
		cc.State = CC_Skills
		return "Распредели 100 пунктов между 10 характеристиками (минимум 4, максимум 18 на каждую).\n\nФормат: WS=XX BS=XX S=XX T=XX I=XX Ag=XX Dex=XX Int=XX WP=XX Fel=XX", true

	default:
		return "Напиши номер варианта (1-3).", false
	}

	cc.Data.TotalXP += cc.Data.XPFromStats
	cc.calculateSecondaryStats()
	cc.State = CC_Skills
	return fmt.Sprintf("Характеристики (бросок 2d10 + бонус расы):\n%s\n\n+ %d XP (всего: %d)\n\n%s",
		cc.getStatsSummary(), cc.Data.XPFromStats, cc.Data.TotalXP, cc.GetPrompt()), true
}

// rollStats generates random characteristics
func (cc *CharacterCreator) rollStats(allowSwap bool) {
	baseStats := []int{
		rand.Intn(10) + rand.Intn(10) + 2, // 2-20
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
		rand.Intn(10) + rand.Intn(10) + 2,
	}

	// Apply race bonuses
	raceBonus := map[string]int{
		"Человек": 30, "Полурослик": 20, "Гном": 30,
		"Высший эльф": 40, "Лесной эльф": 30,
	}

	bonus := raceBonus[cc.Data.Race]
	if bonus == 0 {
		bonus = 30
	}

	// Apply to characteristics
	cc.Data.WS = baseStats[0] + bonus
	cc.Data.BS = baseStats[1] + bonus
	cc.Data.S = baseStats[2] + bonus
	cc.Data.T = baseStats[3] + bonus
	cc.Data.I = baseStats[4] + bonus
	cc.Data.Ag = baseStats[5] + bonus
	cc.Data.Dex = baseStats[6] + bonus
	cc.Data.Int = baseStats[7] + bonus
	cc.Data.WP = baseStats[8] + bonus
	cc.Data.Fel = baseStats[9] + bonus

	// Swap if allowed
	if allowSwap && len(baseStats) > 0 {
		// For simplicity, we'll just note that swap is possible
		// In full implementation, player could reorder
	}
}

// calculateSecondaryStats calculates HP, Fate, etc.
func (cc *CharacterCreator) calculateSecondaryStats() {
	// HP = РС + 2×РВ + РСВ
	rs := cc.Data.S / 10
	rv := cc.Data.T / 10
	rsv := cc.Data.WP / 10

	cc.Data.HP = rs + 2*rv + rsv

	// Fate and Resilience based on race
	fateResilience := map[string][2]int{
		"Человек":     {2, 1},
		"Полурослик":  {0, 2},
		"Гном":        {0, 2},
		"Высший эльф": {0, 0},
		"Лесной эльф": {0, 0},
	}

	fr := fateResilience[cc.Data.Race]
	if len(fr) >= 2 {
		cc.Data.Fate = fr[0]
		cc.Data.Resilience = fr[1]
		cc.Data.Fortune = cc.Data.Fate
		cc.Data.Resolve = cc.Data.Resilience
	}

	// Movement based on race
	movement := map[string]int{
		"Человек": 4, "Полурослик": 3, "Гном": 3, "Высший эльф": 5, "Лесной эльф": 5,
	}
	cc.Data.Movement = movement[cc.Data.Race]
	if cc.Data.Movement == 0 {
		cc.Data.Movement = 4
	}

	// Money based on status
	cc.Data.Money = rand.Intn(10)*2 + cc.Data.StatusLevel*2 // 2d10 * status level
}

// getStatsSummary returns formatted stats
func (cc *CharacterCreator) getStatsSummary() string {
	return fmt.Sprintf(`ББ: %d, ДБ: %d, СС: %d, К: %d
И: %d, Л: %d, О: %d, СТ: %d

HP: %d | Судьба: %d | Упорство: %d | Движение: %d`,
		cc.Data.WS, cc.Data.BS, cc.Data.S, cc.Data.T,
		cc.Data.I, cc.Data.Ag, cc.Data.WP, cc.Data.Fel,
		cc.Data.HP, cc.Data.Fate, cc.Data.Resilience, cc.Data.Movement)
}

// getTalentsList returns talents from race and career
func (cc *CharacterCreator) getTalentsList() string {
	// Simplified - in full version would lookup from rules
	return "Таланты от расы и карьеры:\n(будут добавлены автоматически из правил)"
}

// getGearInfo returns gear info
func (cc *CharacterCreator) getGearInfo() string {
	return fmt.Sprintf("Деньги: %d (по статусу %s %d)\n\nСнаряжение будет добавлено из правил карьеры.",
		cc.Data.Money, cc.Data.Status, cc.Data.StatusLevel)
}

// processAppearance handles appearance generation
func (cc *CharacterCreator) processAppearance(input string) (string, bool) {
	// Generate random appearance
	hairRoll := rand.Intn(20) + 1
	eyeRoll := rand.Intn(20) + 1

	hairColors := []string{"чёрные", "каштановые", "русые", "рыжие", "седые", "белые"}
	eyeColors := []string{"карие", "голубые", "зелёные", "серые", "чёрные"}

	if hairRoll > len(hairColors) {
		hairRoll = len(hairColors)
	}
	if eyeRoll > len(eyeColors) {
		eyeRoll = len(eyeColors)
	}

	cc.Data.HairColor = hairColors[hairRoll-1]
	cc.Data.EyeColor = eyeColors[eyeRoll-1]

	// Age: base + 2d10
	ageBase := map[string]int{"Человек": 18, "Полурослик": 30, "Гном": 40, "Высший эльф": 100, "Лесной эльф": 50}
	base := ageBase[cc.Data.Race]
	if base == 0 {
		base = 18
	}
	cc.Data.Age = base + rand.Intn(20) + 2

	// Height (simplified)
	cc.Data.Height = fmt.Sprintf("%d см", 150+rand.Intn(40))

	cc.State = CC_Personality
	return fmt.Sprintf("Внешность:\n- Волосы: %s\n- Глаза: %s\n- Рост: %s\n- Возраст: %d\n\n%s",
		cc.Data.HairColor, cc.Data.EyeColor, cc.Data.Height, cc.Data.Age, cc.GetPrompt()), true
}

// processPersonality handles personality input
func (cc *CharacterCreator) processPersonality(input string) {
	lines := strings.Split(input, "\n")
	if len(lines) >= 1 {
		cc.Data.Strengths = strings.Split(lines[0], ",")
		for i := range cc.Data.Strengths {
			cc.Data.Strengths[i] = strings.TrimSpace(cc.Data.Strengths[i])
		}
	}
	if len(lines) >= 2 {
		cc.Data.Weaknesses = strings.Split(lines[1], ",")
		for i := range cc.Data.Weaknesses {
			cc.Data.Weaknesses[i] = strings.TrimSpace(cc.Data.Weaknesses[i])
		}
	}
	if len(lines) >= 3 {
		cc.Data.Background = lines[2]
	}
	cc.Data.Motivation = "Стать искателем приключений"
}

// generateReview generates character review
func (cc *CharacterCreator) generateReview() string {
	return fmt.Sprintf(`📋 ПРОВЕРЬ ПЕРСОНАЖА:

**Имя:** %s
**Раса:** %s (+%d XP)
**Карьера:** %s → %s (+%d XP)

**Характеристики:**
ББ: %d, ДБ: %d, СС: %d, К: %d
И: %d, Л: %d, О: %d, СТ: %d

**Вторичные:**
HP: %d | Судьба: %d | Движение: %d

**Внешность:**
Возраст: %d | Рост: %s
Волосы: %s | Глаза: %s

**Характер:**
Сильные: %s
Слабые: %s

**Опыт:** %d всего

Напиши "да" для сохранения или "нет" для отмены.`,
		cc.Data.Name, cc.Data.Race, cc.Data.XPFromRace,
		cc.Data.Class, cc.Data.Career, cc.Data.XPFromCareer,
		cc.Data.WS, cc.Data.BS, cc.Data.S, cc.Data.T,
		cc.Data.I, cc.Data.Ag, cc.Data.WP, cc.Data.Fel,
		cc.Data.HP, cc.Data.Fate, cc.Data.Movement,
		cc.Data.Age, cc.Data.Height, cc.Data.HairColor, cc.Data.EyeColor,
		strings.Join(cc.Data.Strengths, ", "),
		strings.Join(cc.Data.Weaknesses, ", "),
		cc.Data.TotalXP)
}

// saveStep saves current step to markdown file
func (cc *CharacterCreator) saveStep() {
	if cc.Data.BasePath == "" {
		cc.Data.BasePath = "./characters"
	}

	stepNames := map[CharacterCreationState]string{
		CC_Name:        "01_name",
		CC_Race:        "02_race",
		CC_Career:      "03_career",
		CC_Stats:       "04_stats",
		CC_Skills:      "05_skills",
		CC_Talents:     "06_talents",
		CC_Gear:        "07_gear",
		CC_Appearance:  "08_appearance",
		CC_Personality: "09_personality",
		CC_Review:      "10_review",
	}

	stepName, ok := stepNames[cc.State]
	if !ok {
		return
	}

	dir := filepath.Join(cc.Data.BasePath, "creation", cc.Data.Name)
	os.MkdirAll(dir, 0755)

	filename := filepath.Join(dir, stepName+".md")
	content := fmt.Sprintf("# Шаг: %s\n", stepName)
	content += fmt.Sprintf("Время: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Ввод: %s\n\n", cc.currentInput)

	// Add current data snapshot
	content += "\n## Данные персонажа:\n"
	content += fmt.Sprintf("- Имя: %s\n", cc.Data.Name)
	content += fmt.Sprintf("- Раса: %s\n", cc.Data.Race)
	content += fmt.Sprintf("- Карьера: %s\n", cc.Data.Career)

	os.WriteFile(filename, []byte(content), 0644)
}

// GenerateCharacterMarkdown generates final character file
func (cc *CharacterCreator) GenerateCharacterMarkdown() string {
	return fmt.Sprintf(`# %s

**Дата создания:** %s  
**Раса:** %s  
**Карьера:** %s  
**Статус:** %s %d

---

## ХАРАКТЕРИСТИКИ

| Характеристика | Значение | Бонус |
|---|---|---|
| ББ (Боевая Пригодность) | %d | %d |
| ДБ (Дистанция Боя) | %d | %d |
| СС (Сила) | %d | %d |
| К (Классовая/Выносливость) | %d | %d |
| И (Инициатива) | %d | %d |
| Л (Ловкость) | %d | %d |
| О (Общение) | %d | %d |
| СТ (Стойкость) | %d | %d |

**Раны:** %d  
**Движение:** %d

---

## ОПЫТ

| Источник | XP |
|---|---|
| Раса | %d |
| Характеристики | %d |
| Карьера | %d |
| **Итого** | %d |

---

## ВНЕШНОСТЬ

- Возраст: %d
- Рост: %s
- Волосы: %s
- Глаза: %s

---

## ХАРАКТЕР

**Сильные стороны:** %s  
**Слабые стороны:** %s  
**Прошлое:** %s

---

**ПЕРСОНАЖ ГОТОВ К ИГРЕ!**
`,
		cc.Data.Name,
		time.Now().Format("2006-01-02"),
		cc.Data.Race,
		cc.Data.Career,
		cc.Data.Status,
		cc.Data.StatusLevel,
		cc.Data.WS, cc.Data.WS/10,
		cc.Data.BS, cc.Data.BS/10,
		cc.Data.S, cc.Data.S/10,
		cc.Data.T, cc.Data.T/10,
		cc.Data.I, cc.Data.I/10,
		cc.Data.Ag, cc.Data.Ag/10,
		cc.Data.WP, cc.Data.WP/10,
		cc.Data.Fel, cc.Data.Fel/10,
		cc.Data.HP,
		cc.Data.Movement,
		cc.Data.XPFromRace,
		cc.Data.XPFromStats,
		cc.Data.XPFromCareer,
		cc.Data.TotalXP,
		cc.Data.Age,
		cc.Data.Height,
		cc.Data.HairColor,
		cc.Data.EyeColor,
		strings.Join(cc.Data.Strengths, ", "),
		strings.Join(cc.Data.Weaknesses, ", "),
		cc.Data.Background,
	)
}

// SaveToFile saves character to file
func (cc *CharacterCreator) SaveToFile(basePath string) error {
	if basePath == "" {
		basePath = "./characters"
	}

	filename := filepath.Join(basePath, fmt.Sprintf("character_%s.md", cc.Data.Name))
	return os.WriteFile(filename, []byte(cc.GenerateCharacterMarkdown()), 0644)
}

// GetState returns current state
func (cc *CharacterCreator) GetState() CharacterCreationState {
	return cc.State
}

// IsComplete returns true if character creation is complete
func (cc *CharacterCreator) IsComplete() bool {
	return cc.State == CC_Complete || cc.State == CC_Save
}

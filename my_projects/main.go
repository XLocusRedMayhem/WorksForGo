package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------- Модели ----------

type Resource struct {
	Name string `json:"name"`
}

type Source struct {
	Name string `json:"name"`
}

type HistoryEntry struct {
	ID        int     `json:"id"`
	Resource  string  `json:"resource"`
	Source    string  `json:"source"`
	Quantity  float64 `json:"quantity"`
	BuyPrice  float64 `json:"buy_price"`
	CreatedAt string  `json:"created_at"`
}

// Компонент крафта: ресурс или деталь
type CraftIngredient struct {
	Kind string  `json:"kind"` // "resource" | "detail"
	Name string  `json:"name"`
	Qty  float64 `json:"qty"`
}

type Craft struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Ingredients []CraftIngredient `json:"ingredients"`
	BuyPrice    float64           `json:"buy_price"`
	SellPrice   float64           `json:"sell_price"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type Store struct {
	mu        sync.RWMutex
	resources []Resource
	sources   []Source
	history   []HistoryEntry
	crafts    []Craft
	nextHist  int
	nextCraft int
}

const dataDir = "data"

var (
	resourcesFile = filepath.Join(dataDir, "resources.json")
	sourcesFile   = filepath.Join(dataDir, "sources.json")
	historyFile   = filepath.Join(dataDir, "history.json")
	craftsFile    = filepath.Join(dataDir, "crafts.json")
)

// ---------- Загрузка/сохранение ----------

func ensureDir() error {
	return os.MkdirAll(dataDir, 0755)
}

func loadJSON(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func saveJSON(path string, v interface{}) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func NewStore() (*Store, error) {
	if err := ensureDir(); err != nil {
		return nil, err
	}
	s := &Store{}
	if err := loadJSON(resourcesFile, &s.resources); err != nil {
		return nil, err
	}
	if err := loadJSON(sourcesFile, &s.sources); err != nil {
		return nil, err
	}
	if err := loadJSON(historyFile, &s.history); err != nil {
		return nil, err
	}
	if err := loadJSON(craftsFile, &s.crafts); err != nil {
		return nil, err
	}
	maxH, maxC := 0, 0
	for _, h := range s.history {
		if h.ID > maxH {
			maxH = h.ID
		}
	}
	for _, c := range s.crafts {
		if c.ID > maxC {
			maxC = c.ID
		}
	}
	s.nextHist = maxH + 1
	s.nextCraft = maxC + 1
	return s, nil
}

// ---------- Ресурсы ----------

func (s *Store) AddResource(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("название не может быть пустым")
	}
	if len([]rune(name)) > 100 {
		return errors.New("название слишком длинное (максимум 100 символов)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.resources {
		if strings.EqualFold(r.Name, name) {
			return errors.New("ресурс с таким названием уже существует")
		}
	}
	s.resources = append(s.resources, Resource{Name: name})
	sort.Slice(s.resources, func(i, j int) bool {
		return strings.ToLower(s.resources[i].Name) < strings.ToLower(s.resources[j].Name)
	})
	return saveJSON(resourcesFile, s.resources)
}

func (s *Store) DeleteResource(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, r := range s.resources {
		if r.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("ресурс не найден")
	}
	s.resources = append(s.resources[:idx], s.resources[idx+1:]...)
	return saveJSON(resourcesFile, s.resources)
}

// ---------- Источники ----------

func (s *Store) AddSource(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("название не может быть пустым")
	}
	if len([]rune(name)) > 100 {
		return errors.New("название слишком длинное (максимум 100 символов)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.sources {
		if strings.EqualFold(r.Name, name) {
			return errors.New("источник с таким названием уже существует")
		}
	}
	s.sources = append(s.sources, Source{Name: name})
	sort.Slice(s.sources, func(i, j int) bool {
		return strings.ToLower(s.sources[i].Name) < strings.ToLower(s.sources[j].Name)
	})
	return saveJSON(sourcesFile, s.sources)
}

func (s *Store) DeleteSource(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, r := range s.sources {
		if r.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("источник не найден")
	}
	s.sources = append(s.sources[:idx], s.sources[idx+1:]...)
	return saveJSON(sourcesFile, s.sources)
}

// ---------- История ----------

func (s *Store) AddHistory(e HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(e.Resource) == "" {
		return errors.New("выберите ресурс")
	}
	if strings.TrimSpace(e.Source) == "" {
		return errors.New("выберите источник")
	}
	if !s.hasResourceLocked(e.Resource) {
		return errors.New("такого ресурса нет в списке")
	}
	if !s.hasSourceLocked(e.Source) {
		return errors.New("такого источника нет в списке")
	}
	if e.Quantity < 0 || e.BuyPrice < 0 {
		return errors.New("значения не могут быть отрицательными")
	}
	e.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	e.ID = s.nextHist
	s.nextHist++
	s.history = append(s.history, e)
	return saveJSON(historyFile, s.history)
}

func (s *Store) DeleteHistory(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, h := range s.history {
		if h.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("запись не найдена")
	}
	s.history = append(s.history[:idx], s.history[idx+1:]...)
	return saveJSON(historyFile, s.history)
}

func (s *Store) hasResourceLocked(name string) bool {
	for _, r := range s.resources {
		if r.Name == name {
			return true
		}
	}
	return false
}

func (s *Store) hasSourceLocked(name string) bool {
	for _, r := range s.sources {
		if r.Name == name {
			return true
		}
	}
	return false
}

// ---------- Крафты ----------

func (s *Store) AddCraft(c Craft) error {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return errors.New("название крафта не может быть пустым")
	}
	if len([]rune(name)) > 100 {
		return errors.New("название слишком длинное (максимум 100 символов)")
	}
	if c.BuyPrice < 0 || c.SellPrice < 0 {
		return errors.New("стоимость не может быть отрицательной")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// уникальность названия среди крафтов
	for _, cc := range s.crafts {
		if strings.EqualFold(cc.Name, name) {
			return errors.New("крафт с таким названием уже существует")
		}
	}
	// валидация ингредиентов
	if err := s.validateIngredientsLocked(c.Ingredients); err != nil {
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	c.ID = s.nextCraft
	s.nextCraft++
	c.Name = name
	c.CreatedAt = now
	c.UpdatedAt = now
	s.crafts = append(s.crafts, c)
	return saveJSON(craftsFile, s.crafts)
}

func (s *Store) UpdateCraft(id int, c Craft) error {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return errors.New("название крафта не может быть пустым")
	}
	if c.BuyPrice < 0 || c.SellPrice < 0 {
		return errors.New("стоимость не может быть отрицательной")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, cc := range s.crafts {
		if cc.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("крафт не найден")
	}
	for i, cc := range s.crafts {
		if i != idx && strings.EqualFold(cc.Name, name) {
			return errors.New("крафт с таким названием уже существует")
		}
	}
	if err := s.validateIngredientsLocked(c.Ingredients); err != nil {
		return err
	}
	// защита от циклов: деталь не может использовать саму себя
	for _, ing := range c.Ingredients {
		if ing.Kind == "detail" && strings.EqualFold(ing.Name, name) {
			return errors.New("крафт не может содержать сам себя")
		}
	}
	s.crafts[idx].Name = name
	s.crafts[idx].Ingredients = c.Ingredients
	s.crafts[idx].BuyPrice = c.BuyPrice
	s.crafts[idx].SellPrice = c.SellPrice
	s.crafts[idx].UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	return saveJSON(craftsFile, s.crafts)
}

func (s *Store) UpdateCraftPrices(id int, buy, sell float64) error {
	if buy < 0 || sell < 0 {
		return errors.New("стоимость не может быть отрицательной")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.crafts {
		if s.crafts[i].ID == id {
			s.crafts[i].BuyPrice = buy
			s.crafts[i].SellPrice = sell
			s.crafts[i].UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
			return saveJSON(craftsFile, s.crafts)
		}
	}
	return errors.New("крафт не найден")
}

func (s *Store) DeleteCraft(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, c := range s.crafts {
		if c.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("крафт не найден")
	}
	// проверка: используется ли эта деталь в других крафтах
	name := s.crafts[idx].Name
	for _, c := range s.crafts {
		if c.ID == idx+1 { // не совсем корректно, поэтому проверим по ID ниже
		}
		for _, ing := range c.Ingredients {
			if ing.Kind == "detail" && strings.EqualFold(ing.Name, name) && c.ID != s.crafts[idx].ID {
				return fmt.Errorf("деталь используется в крафте «%s»", c.Name)
			}
		}
	}
	s.crafts = append(s.crafts[:idx], s.crafts[idx+1:]...)
	return saveJSON(craftsFile, s.crafts)
}

func (s *Store) validateIngredientsLocked(ings []CraftIngredient) error {
	for _, ing := range ings {
		if strings.TrimSpace(ing.Name) == "" {
			return errors.New("ингредиент без названия")
		}
		if ing.Qty <= 0 {
			return fmt.Errorf("количество ингредиента «%s» должно быть больше 0", ing.Name)
		}
		switch ing.Kind {
		case "resource":
			if !s.hasResourceLocked(ing.Name) {
				return fmt.Errorf("ресурс «%s» не найден", ing.Name)
			}
		case "detail":
			found := false
			for _, c := range s.crafts {
				if strings.EqualFold(c.Name, ing.Name) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("деталь «%s» не найдена", ing.Name)
			}
		default:
			return errors.New("неизвестный тип ингредиента")
		}
	}
	return nil
}

// ---------- Списки (для шаблонов) ----------

func (s *Store) ListResources() []Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Resource, len(s.resources))
	copy(out, s.resources)
	return out
}

func (s *Store) ListSources() []Source {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Source, len(s.sources))
	copy(out, s.sources)
	return out
}

func (s *Store) ListCrafts() []Craft {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Craft, len(s.crafts))
	copy(out, s.crafts)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (s *Store) GetCraft(id int) (Craft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.crafts {
		if c.ID == id {
			return c, true
		}
	}
	return Craft{}, false
}

type HistoryFilter struct {
	Resource string
	Source   string
	From     string
	To       string
}

func (s *Store) ListHistory(f HistoryFilter) []HistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HistoryEntry, 0, len(s.history))
	for _, h := range s.history {
		if f.Resource != "" && h.Resource != f.Resource {
			continue
		}
		if f.Source != "" && h.Source != f.Source {
			continue
		}
		date := ""
		if len(h.CreatedAt) >= 10 {
			date = h.CreatedAt[:10]
		}
		if f.From != "" && date < f.From {
			continue
		}
		if f.To != "" && date > f.To {
			continue
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// ---------- Сводка (пункт 4) ----------

type SummaryRow struct {
	Resource string
	TotalQty float64
	CountQty int
	AvgPrice float64 // средняя цена среди записей с ценой > 0
	SumBuy   float64 // общая стоимость покупки (по всем записям)
}

type Summary struct {
	Rows        []SummaryRow
	GrandQty    float64
	GrandSumBuy float64
	GrandAvg    float64
}

// Логика:
//   - для каждого ресурса считаем суммарное количество по всем записям;
//   - средняя цена покупки = среднеарифметическое среди записей этого ресурса,
//     у которых buy_price > 0 (т.е. «имеют стоимость»);
//   - итоговая стоимость ресурса = средняя цена × суммарное количество
//     (включая записи с нулевой стоимостью);
//   - общая сумма по всем ресурсам = сумма итоговых стоимостей.
func (s *Store) BuildSummary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type acc struct {
		qty      float64
		sumPrice float64
		cntPrice int
		sumBuy   float64
	}
	m := make(map[string]*acc)
	for _, h := range s.history {
		a := m[h.Resource]
		if a == nil {
			a = &acc{}
			m[h.Resource] = a
		}
		a.qty += h.Quantity
		a.sumBuy += h.BuyPrice
		if h.BuyPrice > 0 {
			a.sumPrice += h.BuyPrice
			a.cntPrice++
		}
	}

	res := Summary{}
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	for _, n := range names {
		a := m[n]
		avg := 0.0
		if a.cntPrice > 0 {
			avg = a.sumPrice / float64(a.cntPrice)
		}
		total := avg * a.qty
		res.Rows = append(res.Rows, SummaryRow{
			Resource: n,
			TotalQty: a.qty,
			CountQty: a.cntPrice,
			AvgPrice: avg,
			SumBuy:   total,
		})
		res.GrandQty += a.qty
		res.GrandSumBuy += total
	}
	if len(res.Rows) > 0 {
		res.GrandAvg = res.GrandSumBuy / res.GrandQty
	}
	return res
}

// ---------- HTTP ----------

type PageData struct {
	Resources  []Resource
	Sources    []Source
	Crafts     []Craft
	History    []HistoryEntry
	Filter     HistoryFilter
	Summary    Summary
	EditCraft  *Craft
	Message    string
	IsError    bool
	ActiveTab  string
	TotalQty   float64
	TotalPrice float64
	Count      int
}

type App struct {
	store *Store
	tmpl  *template.Template
}

func (a *App) render(w http.ResponseWriter, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.Execute(w, data); err != nil {
		log.Println("template error:", err)
	}
}

func (a *App) baseData() PageData {
	return PageData{
		Resources: a.store.ListResources(),
		Sources:   a.store.ListSources(),
		Crafts:    a.store.ListCrafts(),
		Summary:   a.store.BuildSummary(),
	}
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	f := HistoryFilter{
		Resource: r.URL.Query().Get("resource"),
		Source:   r.URL.Query().Get("source"),
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
	}
	hist := a.store.ListHistory(f)
	d := a.baseData()
	d.History = hist
	d.Filter = f
	d.Count = len(hist)
	for _, h := range hist {
		d.TotalQty += h.Quantity
		d.TotalPrice += h.BuyPrice
	}
	// редактирование крафта через ?edit=<id>
	if idStr := r.URL.Query().Get("edit"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			if c, ok := a.store.GetCraft(id); ok {
				d.EditCraft = &c
			}
		}
	}
	a.render(w, d)
}

func (a *App) addResource(w http.ResponseWriter, r *http.Request) {
	err := a.store.AddResource(r.FormValue("name"))
	a.respond(w, err, "Ресурс добавлен")
}

func (a *App) deleteResource(w http.ResponseWriter, r *http.Request) {
	err := a.store.DeleteResource(r.FormValue("name"))
	a.respond(w, err, "Ресурс удалён")
}

func (a *App) addSource(w http.ResponseWriter, r *http.Request) {
	err := a.store.AddSource(r.FormValue("name"))
	a.respond(w, err, "Источник добавлен")
}

func (a *App) deleteSource(w http.ResponseWriter, r *http.Request) {
	err := a.store.DeleteSource(r.FormValue("name"))
	a.respond(w, err, "Источник удалён")
}

func (a *App) addHistory(w http.ResponseWriter, r *http.Request) {
	qty, _ := parseNum(r.FormValue("quantity"))
	price, _ := parseNum(r.FormValue("buy_price"))
	err := a.store.AddHistory(HistoryEntry{
		Resource: r.FormValue("resource"),
		Source:   r.FormValue("source"),
		Quantity: qty,
		BuyPrice: price,
	})
	a.respond(w, err, "Запись сохранена")
}

func (a *App) deleteHistory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	err := a.store.DeleteHistory(id)
	a.respond(w, err, "Запись удалена")
}

// ---------- Крафты: обработчики ----------

// Парсит поля формы вида ing_kind[], ing_name[], ing_qty[] (множественные значения)
func parseIngredients(r *http.Request) []CraftIngredient {
	kinds := r.Form["ing_kind"]
	names := r.Form["ing_name"]
	qtys := r.Form["ing_qty"]
	n := len(kinds)
	if len(names) < n {
		n = len(names)
	}
	if len(qtys) < n {
		n = len(qtys)
	}
	var out []CraftIngredient
	for i := 0; i < n; i++ {
		name := strings.TrimSpace(names[i])
		if name == "" {
			continue
		}
		q, _ := parseNum(qtys[i])
		kind := kinds[i]
		if kind != "resource" && kind != "detail" {
			continue
		}
		out = append(out, CraftIngredient{Kind: kind, Name: name, Qty: q})
	}
	return out
}

func (a *App) addCraft(w http.ResponseWriter, r *http.Request) {
	buy, _ := parseNum(r.FormValue("buy_price"))
	sell, _ := parseNum(r.FormValue("sell_price"))
	c := Craft{
		Name:        r.FormValue("name"),
		Ingredients: parseIngredients(r),
		BuyPrice:    buy,
		SellPrice:   sell,
	}
	err := a.store.AddCraft(c)
	a.respond(w, err, "Крафт создан")
}

func (a *App) updateCraft(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	buy, _ := parseNum(r.FormValue("buy_price"))
	sell, _ := parseNum(r.FormValue("sell_price"))
	c := Craft{
		Name:        r.FormValue("name"),
		Ingredients: parseIngredients(r),
		BuyPrice:    buy,
		SellPrice:   sell,
	}
	err := a.store.UpdateCraft(id, c)
	a.respond(w, err, "Крафт обновлён")
}

func (a *App) updateCraftPrices(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	buy, _ := parseNum(r.FormValue("buy_price"))
	sell, _ := parseNum(r.FormValue("sell_price"))
	err := a.store.UpdateCraftPrices(id, buy, sell)
	a.respond(w, err, "Цены обновлены")
}

func (a *App) deleteCraft(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	err := a.store.DeleteCraft(id)
	a.respond(w, err, "Крафт удалён")
}

func (a *App) respond(w http.ResponseWriter, err error, ok string) {
	d := a.baseData()
	d.History = a.store.ListHistory(HistoryFilter{})
	if err != nil {
		d.Message = err.Error()
		d.IsError = true
	} else {
		d.Message = ok
	}
	a.render(w, d)
}

// ---------- Утилиты ----------

func parseNum(s string) (float64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func main() {
	store, err := NewStore()
	if err != nil {
		log.Fatalf("не удалось загрузить данные: %v", err)
	}
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"g": func(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) },
		"hasDetail": func(crafts []Craft, name string) bool {
			for _, c := range crafts {
				if strings.EqualFold(c.Name, name) {
					return true
				}
			}
			return false
		},
	}).Parse(indexHTML)
	if err != nil {
		log.Fatal(err)
	}
	app := &App{store: store, tmpl: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.index)
	mux.HandleFunc("/resources/add", app.addResource)
	mux.HandleFunc("/resources/delete", app.deleteResource)
	mux.HandleFunc("/sources/add", app.addSource)
	mux.HandleFunc("/sources/delete", app.deleteSource)
	mux.HandleFunc("/history/add", app.addHistory)
	mux.HandleFunc("/history/delete", app.deleteHistory)
	mux.HandleFunc("/crafts/add", app.addCraft)
	mux.HandleFunc("/crafts/update", app.updateCraft)
	mux.HandleFunc("/crafts/prices", app.updateCraftPrices)
	mux.HandleFunc("/crafts/delete", app.deleteCraft)

	addr := ":8080"
	fmt.Println("Сервер запущен: http://localhost" + addr)
	fmt.Println("Данные хранятся в папке ./data/")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// ---------- HTML шаблон ----------

const indexHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>Учёт ресурсов и крафтов</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: system-ui, Arial, sans-serif; margin: 0; background: #f4f6f8; color: #222; }
  header { background: #2c3e50; color: #fff; padding: 16px 24px; }
  header h1 { margin: 0; font-size: 20px; }
  .wrap { max-width: 1200px; margin: 24px auto; padding: 0 16px; display: grid; gap: 24px; }
  .two { display: grid; gap: 24px; grid-template-columns: 1fr 1fr; }
  @media (max-width: 800px) { .two { grid-template-columns: 1fr; } }
  .card { background: #fff; border-radius: 10px; padding: 20px; box-shadow: 0 2px 8px rgba(0,0,0,.06); }
  .card h2 { margin: 0 0 14px; font-size: 17px; color: #2c3e50; }
  .card h3 { margin: 18px 0 8px; font-size: 14px; color: #2c3e50; }
  .row { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
  input, select, button, textarea { padding: 8px 10px; border: 1px solid #cbd2d9; border-radius: 6px; font-size: 14px; font-family: inherit; }
  input:focus, select:focus { outline: 2px solid #3498db; border-color: transparent; }
  button { background: #3498db; color: #fff; border: none; cursor: pointer; }
  button:hover { background: #2f88c4; }
  button.danger { background: #e74c3c; }
  button.danger:hover { background: #c0392b; }
  button.small { padding: 4px 8px; font-size: 12px; }
  button.ghost { background: #7f8c8d; }
  button.ghost:hover { background: #6b7a7b; }
  .msg { margin: 12px 0 0; font-size: 13px; padding: 8px 12px; border-radius: 6px; }
  .msg.ok { background: #e8f7ee; color: #1e8449; }
  .msg.err { background: #fdecea; color: #b03a2e; }
  ul.res { list-style: none; padding: 0; margin: 12px 0 0; display: grid; gap: 6px; }
  ul.res li { display: flex; justify-content: space-between; align-items: center;
    padding: 6px 10px; background: #f0f4f8; border-radius: 6px; }
  table { width: 100%; border-collapse: collapse; margin-top: 12px; font-size: 14px; }
  th, td { padding: 8px 6px; text-align: left; border-bottom: 1px solid #e6ecf0; vertical-align: middle; }
  th { background: #eef3f7; font-size: 13px; text-transform: uppercase; color: #555; }
  td.num { text-align: right; }
  .muted { color: #888; font-size: 13px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 10px; align-items: end; }
  label { font-size: 13px; color: #555; display: block; margin-bottom: 4px; }
  .footnote { font-size: 12px; color: #888; margin-top: 8px; }
  .tag { display: inline-block; background: #dfe9f1; color: #2c3e50; padding: 2px 8px;
    border-radius: 12px; font-size: 12px; }
  .tag.detail { background: #f3e6d3; color: #6d4c1e; }
  .summary { display: flex; gap: 24px; flex-wrap: wrap; margin-top: 12px; padding: 10px 14px;
    background: #eef3f7; border-radius: 8px; font-size: 14px; }
  .summary b { color: #2c3e50; }
  .ing-row { display: grid; grid-template-columns: 130px 1fr 110px 40px; gap: 8px; align-items: center; margin-bottom: 8px; }
  .price-form { display: flex; gap: 6px; align-items: center; margin: 0; }
  .price-form input { width: 90px; padding: 4px 6px; font-size: 13px; }
</style>
</head>
<body>
<header><h1>📦 Учёт ресурсов, истории и крафтов</h1></header>

<div class="wrap">

  {{if .Message}}<div class="msg {{if .IsError}}err{{else}}ok{{end}}">{{.Message}}</div>{{end}}

  <!-- 1 и 2 -->
  <div class="two">
    <div class="card">
      <h2>1. Ресурсы</h2>
      <form method="POST" action="/resources/add" class="row">
        <input name="name" placeholder="Название ресурса" required maxlength="100" style="flex:1; min-width:180px">
        <button type="submit">Добавить</button>
      </form>
      <ul class="res">
        {{range .Resources}}
          <li>
            <span>{{.Name}}</span>
            <form method="POST" action="/resources/delete"
                  onsubmit="return confirm('Удалить ресурс &quot;{{.Name}}&quot;?')" style="margin:0">
              <input type="hidden" name="name" value="{{.Name}}">
              <button class="danger small" type="submit">Удалить</button>
            </form>
          </li>
        {{else}}
          <li class="muted" style="justify-content:center">Список пуст</li>
        {{end}}
      </ul>
      <div class="footnote">Дубликаты названий не допускаются (без учёта регистра).</div>
    </div>

    <div class="card">
      <h2>2. Источники (места добычи)</h2>
      <form method="POST" action="/sources/add" class="row">
        <input name="name" placeholder="Название источника" required maxlength="100" style="flex:1; min-width:180px">
        <button type="submit">Добавить</button>
      </form>
      <ul class="res">
        {{range .Sources}}
          <li>
            <span>{{.Name}}</span>
            <form method="POST" action="/sources/delete"
                  onsubmit="return confirm('Удалить источник &quot;{{.Name}}&quot;?')" style="margin:0">
              <input type="hidden" name="name" value="{{.Name}}">
              <button class="danger small" type="submit">Удалить</button>
            </form>
          </li>
        {{else}}
          <li class="muted" style="justify-content:center">Список пуст</li>
        {{end}}
      </ul>
      <div class="footnote">Дубликаты названий не допускаются (без учёта регистра).</div>
    </div>
  </div>

  <!-- 3. Новая запись -->
  <div class="card">
    <h2>3. Новая запись истории добычи</h2>
    <form method="POST" action="/history/add">
      <div class="grid">
        <div>
          <label>Ресурс</label>
          <select name="resource" required style="width:100%">
            <option value="">— выберите —</option>
            {{range .Resources}}<option value="{{.Name}}">{{.Name}}</option>{{end}}
          </select>
        </div>
        <div>
          <label>Источник (место добычи)</label>
          <select name="source" required style="width:100%">
            <option value="">— выберите —</option>
            {{range .Sources}}<option value="{{.Name}}">{{.Name}}</option>{{end}}
          </select>
        </div>
        <div>
          <label>Количество (0 разрешено)</label>
          <input name="quantity" type="number" step="any" min="0" value="0" required style="width:100%">
        </div>
        <div>
          <label>Стоимость покупки (0 разрешено)</label>
          <input name="buy_price" type="number" step="any" min="0" value="0" required style="width:100%">
        </div>
        <div>
          <button type="submit" style="width:100%">Сохранить</button>
        </div>
      </div>
    </form>
    <div class="footnote">Дата и время проставляются сервером (до секунд).</div>
  </div>

  <!-- 4. Сводка -->
  <div class="card">
    <h2>4. Общая сумма ресурсов</h2>
    {{if .Summary.Rows}}
    <div class="summary">
      <span>Σ количество: <b>{{g .Summary.GrandQty}}</b></span>
      <span>Σ стоимость: <b>{{g .Summary.GrandSumBuy}}</b></span>
      <span>Средняя цена: <b>{{g .Summary.GrandAvg}}</b></span>
    </div>
    <table>
      <thead>
        <tr>
          <th>Ресурс</th>
          <th class="num">Всего количества</th>
          <th class="num">Записей с ценой</th>
          <th class="num">Средняя цена покупки</th>
          <th class="num">Итоговая стоимость</th>
        </tr>
      </thead>
      <tbody>
        {{range .Summary.Rows}}
        <tr>
          <td>{{.Resource}}</td>
          <td class="num">{{g .TotalQty}}</td>
          <td class="num">{{.CountQty}}</td>
          <td class="num">{{g .AvgPrice}}</td>
          <td class="num">{{g .SumBuy}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    <div class="footnote">
      Средняя цена = среднеарифметическое среди записей с ценой &gt; 0.<br>
      Итоговая стоимость = средняя цена × общее количество (включая записи с нулевой стоимостью).
    </div>
    {{else}}
      <div class="muted">Нет данных для сводки.</div>
    {{end}}
  </div>

  <!-- 5. История + фильтр -->
  <div class="card">
    <h2>5. История добычи</h2>

    <form method="GET" action="/">
      <div class="grid">
        <div>
          <label>Ресурс</label>
          <select name="resource" style="width:100%">
            <option value="">— все —</option>
            {{range .Resources}}
              <option value="{{.Name}}" {{if eq .Name $.Filter.Resource}}selected{{end}}>{{.Name}}</option>
            {{end}}
          </select>
        </div>
        <div>
          <label>Источник</label>
          <select name="source" style="width:100%">
            <option value="">— все —</option>
            {{range .Sources}}
              <option value="{{.Name}}" {{if eq .Name $.Filter.Source}}selected{{end}}>{{.Name}}</option>
            {{end}}
          </select>
        </div>
        <div>
          <label>Дата с</label>
          <input name="from" type="date" value="{{.Filter.From}}" style="width:100%">
        </div>
        <div>
          <label>Дата по</label>
          <input name="to" type="date" value="{{.Filter.To}}" style="width:100%">
        </div>
        <div class="row">
          <button type="submit">Фильтровать</button>
          <a href="/" class="muted">Сбросить</a>
        </div>
      </div>
    </form>

    {{if .History}}
    <div class="summary">
      <span>Записей: <b>{{.Count}}</b></span>
      <span>Σ количество: <b>{{g .TotalQty}}</b></span>
      <span>Σ стоимость покупки: <b>{{g .TotalPrice}}</b></span>
    </div>
    <table>
      <thead>
        <tr>
          <th>ID</th><th>Ресурс</th><th>Источник</th>
          <th class="num">Кол-во</th><th class="num">Стоимость</th>
          <th>Дата и время</th><th></th>
        </tr>
      </thead>
      <tbody>
        {{range .History}}
        <tr>
          <td>{{.ID}}</td>
          <td>{{.Resource}}</td>
          <td><span class="tag">{{.Source}}</span></td>
          <td class="num">{{g .Quantity}}</td>
          <td class="num">{{g .BuyPrice}}</td>
          <td>{{.CreatedAt}}</td>
          <td>
            <form method="POST" action="/history/delete"
                  onsubmit="return confirm('Удалить запись #{{.ID}}?')" style="margin:0">
              <input type="hidden" name="id" value="{{.ID}}">
              <button class="danger small" type="submit">Удалить</button>
            </form>
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{else}}
      <div class="muted" style="margin-top:12px">Записей нет.</div>
    {{end}}
  </div>

  <!-- 6. Крафты -->
  <div class="card">
    <h2>6. Крафты деталей</h2>

    {{if .EditCraft}}
      <!-- Редактирование -->
      <form method="POST" action="/crafts/update" style="margin-bottom:16px; padding:14px; background:#f9fbfc; border:1px solid #e1e8ee; border-radius:8px;">
        <h3>✏️ Редактирование крафта #{{.EditCraft.ID}}</h3>
        <input type="hidden" name="id" value="{{.EditCraft.ID}}">
        <div class="grid">
          <div>
            <label>Название детали</label>
            <input name="name" value="{{.EditCraft.Name}}" required maxlength="100" style="width:100%">
          </div>
          <div>
            <label>Стоимость покупки</label>
            <input name="buy_price" type="number" step="any" min="0" value="{{g .EditCraft.BuyPrice}}" style="width:100%">
          </div>
          <div>
            <label>Стоимость продажи</label>
            <input name="sell_price" type="number" step="any" min="0" value="{{g .EditCraft.SellPrice}}" style="width:100%">
          </div>
        </div>

        <h3>Ингредиенты</h3>
        {{range .EditCraft.Ingredients}}
          <div class="ing-row">
            <select name="ing_kind">
              <option value="resource" {{if eq .Kind "resource"}}selected{{end}}>Ресурс</option>
              <option value="detail" {{if eq .Kind "detail"}}selected{{end}}>Деталь</option>
            </select>
            <input name="ing_name" value="{{.Name}}" placeholder="Название" required>
            <input name="ing_qty" type="number" step="any" min="0.001" value="{{g .Qty}}" required>
            <span></span>
          </div>
        {{end}}
        <div class="ing-row">
          <select name="ing_kind">
            <option value="resource">Ресурс</option>
            <option value="detail">Деталь</option>
          </select>
          <input name="ing_name" placeholder="Название (можно пусто)">
          <input name="ing_qty" type="number" step="any" min="0" value="1">
          <span></span>
        </div>
        <div class="row" style="margin-top:8px">
          <button type="submit">Сохранить изменения</button>
          <a href="/" class="muted">Отменить</a>
        </div>
      </form>
    {{end}}

    <!-- Создание -->
    <form method="POST" action="/crafts/add">
      <div class="grid">
        <div>
          <label>Название детали</label>
          <input name="name" required maxlength="100" style="width:100%">
        </div>
        <div>
          <label>Стоимость покупки</label>
          <input name="buy_price" type="number" step="any" min="0" value="0" style="width:100%">
        </div>
        <div>
          <label>Стоимость продажи</label>
          <input name="sell_price" type="number" step="any" min="0" value="0" style="width:100%">
        </div>
      </div>

      <h3>Ингредиенты (могут отсутствовать)</h3>
      <div class="ing-row">
        <select name="ing_kind">
          <option value="resource">Ресурс</option>
          <option value="detail">Деталь</option>
        </select>
        <input name="ing_name" placeholder="Название">
        <input name="ing_qty" type="number" step="any" min="0" value="1">
        <span></span>
      </div>
      <div class="ing-row">
        <select name="ing_kind">
          <option value="resource">Ресурс</option>
          <option value="detail">Деталь</option>
        </select>
        <input name="ing_name" placeholder="Название">
        <input name="ing_qty" type="number" step="any" min="0" value="1">
        <span></span>
      </div>
      <div class="ing-row">
        <select name="ing_kind">
          <option value="resource">Ресурс</option>
          <option value="detail">Деталь</option>
        </select>
        <input name="ing_name" placeholder="Название">
        <input name="ing_qty" type="number" step="any" min="0" value="1">
        <span></span>
      </div>
      <div class="row" style="margin-top:10px">
        <button type="submit">Создать крафт</button>
      </div>
      <div class="footnote">
        Пустые строки игнорируются. Ингредиенты могут ссылаться на уже существующие крафты (детали).
      </div>
    </form>

    <h3>Список крафтов</h3>
    {{if .Crafts}}
    <table>
      <thead>
        <tr>
          <th>ID</th><th>Название</th><th>Ингредиенты</th>
          <th class="num">Покупка</th><th class="num">Продажа</th>
          <th>Обновлён</th><th></th>
        </tr>
      </thead>
      <tbody>
        {{range .Crafts}}
        <tr>
          <td>{{.ID}}</td>
          <td>{{.Name}}</td>
          <td>
            {{if .Ingredients}}
              {{range .Ingredients}}
                <span class="tag {{if eq .Kind "detail"}}detail{{end}}">{{.Name}} × {{g .Qty}}</span>
              {{end}}
            {{else}}
              <span class="muted">—</span>
            {{end}}
          </td>
          <td class="num">
            <form method="POST" action="/crafts/prices" class="price-form">
              <input type="hidden" name="id" value="{{.ID}}">
              <input name="buy_price" type="number" step="any" min="0" value="{{g .BuyPrice}}">
              <input name="sell_price" type="number" step="any" min="0" value="{{g .SellPrice}}">
              <button class="small" type="submit" title="Обновить цены">💾</button>
            </form>
          </td>
          <td></td>
          <td>{{.UpdatedAt}}</td>
          <td class="row" style="gap:4px">
            <a href="/?edit={{.ID}}" class="muted">Изменить</a>
            <form method="POST" action="/crafts/delete"
                  onsubmit="return confirm('Удалить крафт &quot;{{.Name}}&quot;?')" style="margin:0">
              <input type="hidden" name="id" value="{{.ID}}">
              <button class="danger small" type="submit">Удалить</button>
            </form>
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{else}}
      <div class="muted" style="margin-top:8px">Крафтов пока нет.</div>
    {{end}}
  </div>

</div>
</body>
</html>
`

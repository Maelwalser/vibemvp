package arch

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Node / Edge types ─────────────────────────────────────────────────────────

type archNodeKind int

const (
	archFrontend archNodeKind = iota
	archAPIGateway
	archService
	archBroker
	archDatabase
	archFileStorage // shares column 4 with databases (stacked)
	archExternalAPI
)

type archNode struct {
	id          string
	kind        archNodeKind
	label       string
	environment string
	configRef   string // references BackendPillar.StackConfigs[*].Name
}

// archEdge direction constants — mirror CommLink.Direction values.
const (
	dirUnidirectional = "unidirectional"
	dirBidirectional  = "bidirectional"
)

type archEdge struct {
	id        string // "edge.N" — unique index-based ID
	fromID    string
	toID      string
	label     string
	direction string
}

// edgeBounds is the bounding rectangle of a drawn arrow on the canvas.
type edgeBounds struct {
	x1, x2, y1, y2 int
}

// archEnvGroup is a bounding rect drawn around nodes sharing an environment or config.
type archEnvGroup struct {
	label string
	color string
	x, y  int
	w, h  int
}

// archDiagramData holds the raw render output plus metadata for colorization.
type archDiagramData struct {
	rawLines         []string
	nodePositions    map[string]nodePos
	edgeBoundsMap    map[string]edgeBounds
	edgePathMap      map[string][]edgeBounds // precise path segments for per-pixel blink
	envGroups        []archEnvGroup
	frontendNodes    []archNode
	gatewayNodes     []archNode
	serviceNodes     []archNode
	brokerNodes      []archNode
	dbNodes          []archNode
	fsNodes          []archNode // file-storage nodes, stacked below dbNodes
	extNodes         []archNode
	selectedID       string
	infoColorRanges  []colorRange // priority-4 colors for the info panel
	labelColorRanges []colorRange // priority-2 colors for column/section header labels
}

// AnimFramePtr points to the global animation frame counter owned by the
// parent ui package. It is set via SetAnimFramePtr at initialization.
var AnimFramePtr *int

// SetAnimFramePtr sets the pointer to the global animation frame counter.
func SetAnimFramePtr(p *int) { AnimFramePtr = p }

// animFrame returns the current animation frame value (0 if unset).
func animFrame() int {
	if AnimFramePtr != nil {
		return *AnimFramePtr
	}
	return 0
}

// ── Screen state ──────────────────────────────────────────────────────────────

// Screen is the full-screen architecture overview overlay.
type Screen struct {
	nodes          []archNode
	edges          []archEdge
	nodeInfo       map[string][]string // node or edge id → detail lines shown in info panel
	envInfoMap     map[string]string   // env name → "ComputeEnv · CloudProvider"
	configLabelMap map[string]string   // ConfigRef name → "Language · Framework"
	selectedID     string
	scrollY        int
	scrollX        int
	wantsQuit      bool
}

// NewScreen creates a zero-value Screen.
func NewScreen() Screen { return Screen{} }

// Open rebuilds the graph from the given manifest and resets scroll.
func (s Screen) Open(mf *manifest.Manifest) Screen {
	nodes, edges := buildArchGraph(mf)
	firstID := ""
	if len(nodes) > 0 {
		firstID = nodes[0].id
	}
	return Screen{
		nodes:          nodes,
		edges:          edges,
		nodeInfo:       buildAllInfo(mf, nodes, edges),
		envInfoMap:     buildEnvInfoMap(mf),
		configLabelMap: buildConfigLabelMap(mf),
		selectedID:     firstID,
	}
}

func (s Screen) WantsQuit() bool { return s.wantsQuit }

func (s Screen) HintLine() string {
	hints := []string{
		core.StyleHelpKey.Render("h/l") + core.StyleHelpDesc.Render(" navigate"),
		core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" cycle"),
		core.StyleHelpKey.Render("c") + core.StyleHelpDesc.Render(" comm links"),
		core.StyleHelpKey.Render("H/L") + core.StyleHelpDesc.Render(" pan"),
		core.StyleHelpKey.Render("g") + core.StyleHelpDesc.Render(" reset"),
		core.StyleHelpKey.Render("q") + core.StyleHelpDesc.Render(" close"),
	}
	sep := core.StyleHelpDesc.Render("  │  ")
	return "  " + strings.Join(hints, sep)
}

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch key.String() {
	case "q", "esc", "P":
		s.wantsQuit = true
	case "l":
		s = s.navigateRight()
	case "h":
		s = s.navigateLeft()
	case "j", "down":
		s = s.navigateDown()
	case "k", "up":
		s = s.navigateUp()
	case "c":
		s = s.selectCommEdge()
	case "L": // Shift+L — pan right
		s.scrollX++
	case "H": // Shift+H — pan left
		if s.scrollX > 0 {
			s.scrollX--
		}
	case "g":
		s.scrollY = 0
		s.scrollX = 0
	}
	return s, nil
}

// View renders the diagram. Pipeline: build raw canvas → clip plain text → colorize.
// Clipping on plain text avoids ANSI-code rune inflation that hides right-hand columns.
func (s Screen) View(w, h int) string {
	data := buildRawArchDiagram(s.nodes, s.edges, s.nodeInfo, s.envInfoMap, s.configLabelMap, s.selectedID, w)
	rawLines := data.rawLines
	total := len(rawLines)

	sy := s.autoScrollY(data, s.scrollY, h)
	sx := s.scrollX

	maxScrollY := total - h
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	if sy > maxScrollY {
		sy = maxScrollY
	}

	// Clip raw lines — rune count is accurate on plain text.
	clipped := make([]string, 0, h)
	for i := sy; i < sy+h && i < total; i++ {
		runes := []rune(rawLines[i])
		if sx < len(runes) {
			runes = runes[sx:]
		} else {
			runes = nil
		}
		if len(runes) > w {
			runes = runes[:w]
		}
		clipped = append(clipped, string(runes))
	}
	for len(clipped) < h {
		clipped = append(clipped, "")
	}

	return colorizeClipped(clipped, data, sx, sy)
}

// autoScrollY returns a scrollY that keeps the selected item inside the viewport.
func (s Screen) autoScrollY(data archDiagramData, currentScroll, viewH int) int {
	if strings.HasPrefix(s.selectedID, "edge.") {
		if eb, ok := data.edgeBoundsMap[s.selectedID]; ok {
			midYPos := (eb.y1 + eb.y2) / 2
			if midYPos < currentScroll {
				return midYPos
			}
			if midYPos >= currentScroll+viewH {
				v := midYPos - viewH/2
				if v < 0 {
					v = 0
				}
				return v
			}
		}
		return currentScroll
	}
	if p, ok := data.nodePositions[s.selectedID]; ok {
		nodeBottom := p.y + p.h
		if nodeBottom > currentScroll+viewH {
			return nodeBottom - viewH
		}
		if p.y < currentScroll {
			return p.y
		}
	}
	return currentScroll
}

// ── Navigation helpers ────────────────────────────────────────────────────────

func nodeColumn(kind archNodeKind) int {
	switch kind {
	case archFrontend:
		return 0
	case archAPIGateway:
		return 1
	case archService:
		return 2
	case archBroker:
		return 3
	case archDatabase, archFileStorage, archExternalAPI:
		return 4 // databases, file storages and external APIs share the data column
	}
	return -1
}

func (s Screen) nodeColumnByID(id string) int {
	for _, n := range s.nodes {
		if n.id == id {
			return nodeColumn(n.kind)
		}
	}
	return 0
}

func (s Screen) nodesInColumn(col int) []archNode {
	var result []archNode
	for _, n := range s.nodes {
		if nodeColumn(n.kind) == col {
			result = append(result, n)
		}
	}
	return result
}

func (s Screen) selectedEdge() (archEdge, bool) {
	for _, e := range s.edges {
		if e.id == s.selectedID {
			return e, true
		}
	}
	return archEdge{}, false
}

// siblingEdges returns all edges that share the same column-pair as e.
func (s Screen) siblingEdges(e archEdge) []archEdge {
	fromCol := s.nodeColumnByID(e.fromID)
	toCol := s.nodeColumnByID(e.toID)
	var result []archEdge
	for _, edge := range s.edges {
		if s.nodeColumnByID(edge.fromID) == fromCol && s.nodeColumnByID(edge.toID) == toCol {
			result = append(result, edge)
		}
	}
	return result
}

// navigateRight: node → select outgoing edge; edge → go to target node.
func (s Screen) navigateRight() Screen {
	if e, ok := s.selectedEdge(); ok {
		s.selectedID = e.toID
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	for _, e := range s.edges {
		if e.fromID == s.selectedID && s.nodeColumnByID(e.toID) > col {
			s.selectedID = e.id
			return s
		}
	}
	// No direct edge — jump to first node in next non-empty column (skip gaps)
	for nextCol := col + 1; nextCol <= 4; nextCol++ {
		for _, n := range s.nodes {
			if nodeColumn(n.kind) == nextCol {
				s.selectedID = n.id
				return s
			}
		}
	}
	return s
}

// navigateLeft: node → select incoming edge; edge → go to source node.
func (s Screen) navigateLeft() Screen {
	if e, ok := s.selectedEdge(); ok {
		s.selectedID = e.fromID
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	for _, e := range s.edges {
		if e.toID == s.selectedID && s.nodeColumnByID(e.fromID) < col {
			s.selectedID = e.id
			return s
		}
	}
	// Jump to first node in previous non-empty column (skip gaps)
	for prevCol := col - 1; prevCol >= 0; prevCol-- {
		for _, n := range s.nodes {
			if nodeColumn(n.kind) == prevCol {
				s.selectedID = n.id
				return s
			}
		}
	}
	return s
}

// navigateDown: cycle to next node in column, or next sibling edge.
func (s Screen) navigateDown() Screen {
	if e, ok := s.selectedEdge(); ok {
		siblings := s.siblingEdges(e)
		for i, sib := range siblings {
			if sib.id == e.id {
				s.selectedID = siblings[(i+1)%len(siblings)].id
				return s
			}
		}
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	nodes := s.nodesInColumn(col)
	for i, n := range nodes {
		if n.id == s.selectedID {
			s.selectedID = nodes[(i+1)%len(nodes)].id
			return s
		}
	}
	return s
}

// navigateUp: cycle to previous node in column, or previous sibling edge.
func (s Screen) navigateUp() Screen {
	if e, ok := s.selectedEdge(); ok {
		siblings := s.siblingEdges(e)
		for i, sib := range siblings {
			if sib.id == e.id {
				s.selectedID = siblings[(i-1+len(siblings))%len(siblings)].id
				return s
			}
		}
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	nodes := s.nodesInColumn(col)
	for i, n := range nodes {
		if n.id == s.selectedID {
			s.selectedID = nodes[(i-1+len(nodes))%len(nodes)].id
			return s
		}
	}
	return s
}

// selectCommEdge selects outgoing svc→svc communication edges from the current service.
// If already on a same-column edge, it cycles to the next sibling.
func (s Screen) selectCommEdge() Screen {
	// If already on a same-column edge, cycle to next sibling.
	if e, ok := s.selectedEdge(); ok {
		fromCol := s.nodeColumnByID(e.fromID)
		toCol := s.nodeColumnByID(e.toID)
		if fromCol == toCol {
			siblings := s.siblingEdges(e)
			for i, sib := range siblings {
				if sib.id == e.id {
					s.selectedID = siblings[(i+1)%len(siblings)].id
					return s
				}
			}
		}
		return s
	}
	// Find first outgoing svc→svc edge from the current service node.
	col := s.nodeColumnByID(s.selectedID)
	for _, e := range s.edges {
		if e.fromID == s.selectedID && s.nodeColumnByID(e.toID) == col {
			s.selectedID = e.id
			return s
		}
	}
	return s
}

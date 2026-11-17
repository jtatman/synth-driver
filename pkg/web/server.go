package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/james-see/imagination-engine/pkg/db"
	"github.com/james-see/imagination-engine/pkg/midiout"
	"github.com/james-see/imagination-engine/pkg/seq"
	"github.com/james-see/imagination-engine/pkg/theory"
)

type Server struct {
	store  *db.Store
	engine *midiout.Engine
}

func NewServer(store *db.Store) *Server {
	return &Server{
		store:  store,
		engine: midiout.NewEngine(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/progressions", s.handleListProgressions)
	mux.HandleFunc("/api/progression", s.handleGetProgression)
	mux.HandleFunc("/api/generate", s.handleGenerateHybrid)
	mux.HandleFunc("/api/export/midi", s.handleExportMIDI)
	mux.HandleFunc("/api/export/crave-seq", s.handleExportCraveSeq)
	mux.HandleFunc("/api/export/td3-seq", s.handleExportTD3Seq)
	mux.HandleFunc("/api/export/svg", s.handleExportSVG)
	mux.HandleFunc("/api/midi/ports", s.handleMIDIPorts)
	mux.HandleFunc("/api/midi/send", s.handleMIDISend)
	mux.HandleFunc("/api/transport/sync", s.handleTransportSync)
	mux.HandleFunc("/api/transport/stop", s.handleTransportStop)
	mux.HandleFunc("/api/transport/status", s.handleTransportStatus)
	return mux
}

func parseVoiceSettingsFromQuery(r *http.Request) theory.VoiceSettings {
	q := r.URL.Query()
	vs := theory.DefaultVoiceSettings()
	if v, err := strconv.Atoi(q.Get("crave_oct")); err == nil && v >= -3 && v <= 3 {
		vs.CraveOctave = v
	}
	if v, err := strconv.Atoi(q.Get("east_oct")); err == nil && v >= -3 && v <= 3 {
		vs.EastBeastOctave = v
	}
	if v, err := strconv.Atoi(q.Get("edge_oct")); err == nil && v >= -3 && v <= 3 {
		vs.EdgeOctave = v
	}
	if v, err := strconv.Atoi(q.Get("poly_oct")); err == nil && v >= -3 && v <= 3 {
		vs.PolyOctave = v
	}
	if v, err := strconv.Atoi(q.Get("crave_ch")); err == nil && v >= 0 && v <= 15 {
		vs.CraveChannel = v
	}
	if v, err := strconv.Atoi(q.Get("east_ch")); err == nil && v >= 0 && v <= 15 {
		vs.EastBeastChannel = v
	}
	if v, err := strconv.Atoi(q.Get("edge_ch")); err == nil && v >= 0 && v <= 15 {
		vs.EdgeChannel = v
	}
	if v, err := strconv.Atoi(q.Get("poly_ch")); err == nil && v >= 0 && v <= 15 {
		vs.PolyChannel = v
	}
	if p := q.Get("poly_port"); p != "" {
		vs.PolyPortID = p
	}
	if p := q.Get("crave_port"); p != "" {
		vs.CravePortID = p
	}
	if p := q.Get("east_port"); p != "" {
		vs.EastBeastPortID = p
	}
	if p := q.Get("edge_port"); p != "" {
		vs.EdgePortID = p
	}
	return vs
}

func (s *Server) resolveArrangement(r *http.Request) (theory.Arrangement, error) {
	q := r.URL.Query()
	mode := q.Get("source")
	key := q.Get("key")
	profile := q.Get("profile")
	if profile != "v1" && profile != "v2" && profile != "v3" {
		profile = theory.DefaultEngineProfile()
	}
	comp, _ := strconv.ParseFloat(q.Get("complexity"), 64)
	colorD, _ := strconv.ParseFloat(q.Get("color"), 64)
	seed, _ := strconv.ParseInt(q.Get("seed"), 10, 64)
	if seed == 0 {
		seed = 42
	}

	var arr theory.Arrangement
	if mode == "hybrid" {
		g1 := q.Get("primary")
		g2 := q.Get("secondary")
		if comp <= 0 {
			comp = 0.60
		}
		if colorD <= 0 {
			colorD = 0.70
		}
		arr = theory.GenerateHybridProgressionWithProfile(g1, g2, key, comp, colorD, seed, profile)
	} else {
		id, _ := strconv.Atoi(q.Get("id"))
		if id <= 0 {
			id = 1
		}
		var err error
		arr, err = s.store.GetProgression(id)
		if err != nil {
			return arr, err
		}
		arr.Profile = profile
		if comp > 0 {
			arr.Complexity = comp
		}
		if colorD > 0 {
			arr.ColorDensity = colorD
		}
		arr = theory.TransposeArrangement(arr, key, seed)
	}
	if bpmOverride, err := strconv.Atoi(q.Get("bpm")); err == nil && bpmOverride >= 30 && bpmOverride <= 300 {
		arr.BPM = bpmOverride
	}
	vs := parseVoiceSettingsFromQuery(r)
	arr = theory.ApplyVoiceSettings(arr, vs)
	return arr, nil
}

type transportSyncRequest struct {
	Playing     bool                 `json:"playing"`
	PortID      string               `json:"port_id"`
	SongMode    string               `json:"song_mode"`
	SliceIdx    int                  `json:"slice_idx"`
	StageIdx    int                  `json:"stage_idx"`
	BPM         int                  `json:"bpm"`
	SendClock   bool                 `json:"send_clock"`
	ClockDiv    int                  `json:"clock_div"`
	SendModCC   bool                 `json:"send_mod_cc"`
	EnablePoly  bool                 `json:"enable_poly"`
	EnableCrave bool                 `json:"enable_crave"`
	EnableEast  bool                 `json:"enable_east"`
	EnableEdge  bool                 `json:"enable_edge"`
	VoiceConfig theory.VoiceSettings `json:"voice_config"`
	Arrangement theory.Arrangement   `json:"arrangement"`
}

func (s *Server) handleTransportSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req transportSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.engine.UpdateAndStart(midiout.TransportState{
		Playing:     req.Playing,
		PortID:      req.PortID,
		SongMode:    req.SongMode,
		SliceIdx:    req.SliceIdx,
		StageIdx:    req.StageIdx,
		BPM:         req.BPM,
		SendClock:   req.SendClock,
		ClockDiv:    req.ClockDiv,
		SendModCC:   req.SendModCC,
		EnablePoly:  req.EnablePoly,
		EnableCrave: req.EnableCrave,
		EnableEast:  req.EnableEast,
		EnableEdge:  req.EnableEdge,
		VoiceConfig: req.VoiceConfig,
		Arrangement: req.Arrangement,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTransportStop(w http.ResponseWriter, r *http.Request) {
	s.engine.Stop()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTransportStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.engine.Status())
}

type midiSendRequest struct {
	PortID  string `json:"port_id"`
	Raw     []byte `json:"raw"`
	Channel int    `json:"channel"`
	Note    int    `json:"note"`
	Vel     int    `json:"vel"`
	DurMs   int    `json:"dur_ms"`
}

func (s *Server) handleMIDIPorts(w http.ResponseWriter, r *http.Request) {
	ports, err := midiout.ListHardwarePorts()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"ports":           []midiout.PortInfo{},
			"default_profile": theory.DefaultEngineProfile(),
			"error":           err.Error(),
		})
		return
	}
	if ports == nil {
		ports = []midiout.PortInfo{}
	}
	json.NewEncoder(w).Encode(map[string]any{
		"ports":           ports,
		"default_profile": theory.DefaultEngineProfile(),
	})
}

func (s *Server) handleMIDISend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req midiSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.PortID == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if len(req.Raw) > 0 {
		if err := midiout.SendHardwareMIDI(req.PortID, req.Raw); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if req.Raw[0] == 0xFC {
			for ch := byte(0); ch < 16; ch++ {
				_ = midiout.SendHardwareMIDI(req.PortID, []byte{0xB0 | ch, 123, 0})
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ch := byte(req.Channel & 0x0F)
	note := byte(req.Note & 0x7F)
	vel := byte(req.Vel & 0x7F)
	if err := midiout.SendHardwareMIDI(req.PortID, []byte{0x90 | ch, note, vel}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dur := req.DurMs
	if dur <= 0 {
		dur = 100
	}
	portID := req.PortID
	go func() {
		time.Sleep(time.Duration(dur) * time.Millisecond)
		_ = midiout.SendHardwareMIDI(portID, []byte{0x80 | ch, note, 0})
	}()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListProgressions(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	search := r.URL.Query().Get("q")
	targetKey := strings.TrimSpace(r.URL.Query().Get("key"))
	list, err := s.store.ListProgressions(genre, search)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if targetKey != "" {
		for i := range list {
			list[i] = theory.TransposeArrangement(list[i], targetKey, int64(list[i].ID*101+7))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleGetProgression(w http.ResponseWriter, r *http.Request) {
	arr, err := s.resolveArrangement(r)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arr)
}

func (s *Server) handleGenerateHybrid(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	g1 := q.Get("primary")
	g2 := q.Get("secondary")
	key := q.Get("key")
	profile := q.Get("profile")
	if profile != "v1" && profile != "v2" && profile != "v3" {
		profile = theory.DefaultEngineProfile()
	}
	comp, _ := strconv.ParseFloat(q.Get("complexity"), 64)
	colorD, _ := strconv.ParseFloat(q.Get("color"), 64)
	seed, _ := strconv.ParseInt(q.Get("seed"), 10, 64)
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	if comp <= 0 {
		comp = 0.60
	}
	if colorD <= 0 {
		colorD = 0.70
	}
	arr := theory.GenerateHybridProgressionWithProfile(g1, g2, key, comp, colorD, seed, profile)
	if bpmOverride, err := strconv.Atoi(q.Get("bpm")); err == nil && bpmOverride >= 30 && bpmOverride <= 300 {
		arr.BPM = bpmOverride
	}
	vs := parseVoiceSettingsFromQuery(r)
	arr = theory.ApplyVoiceSettings(arr, vs)
	if q.Get("save") == "1" {
		id, err := s.store.InsertProgression(arr, "hybrid-generator")
		if err == nil {
			arr.ID = int(id)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arr)
}

func (s *Server) handleExportMIDI(w http.ResponseWriter, r *http.Request) {
	arr, err := s.resolveArrangement(r)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	data := midiout.EncodeMultiTrackMIDI(arr)
	fname := slugify(arr.Title+"_"+arr.RootKey+"_"+arr.Profile) + ".mid"
	w.Header().Set("Content-Type", "audio/midi")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	w.Write(data)
}

func (s *Server) handleExportCraveSeq(w http.ResponseWriter, r *http.Request) {
	arr, err := s.resolveArrangement(r)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	sliceIdx, _ := strconv.Atoi(r.URL.Query().Get("slice"))
	if sliceIdx < 0 || sliceIdx >= len(arr.Slices) {
		sliceIdx = 0
	}
	voice := r.URL.Query().Get("voice")
	sl := arr.Slices[sliceIdx]
	steps := sl.CraveSteps
	suffix := "crave"
	if voice == "eastbeast" {
		steps = sl.EastBeastSteps
		suffix = "eastbeast_crave"
	}
	data := seq.EncodeCraveSeq(steps, sl.Swing)
	fname := fmt.Sprintf("%s_%s_%s.seq", slugify(arr.Title+"_"+arr.RootKey), sl.Slug, suffix)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	w.Write(data)
}

func (s *Server) handleExportTD3Seq(w http.ResponseWriter, r *http.Request) {
	arr, err := s.resolveArrangement(r)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	sliceIdx, _ := strconv.Atoi(r.URL.Query().Get("slice"))
	if sliceIdx < 0 || sliceIdx >= len(arr.Slices) {
		sliceIdx = 0
	}
	sl := arr.Slices[sliceIdx]
	data := seq.EncodeTD3Seq(sl.CraveSteps)
	fname := fmt.Sprintf("%s_%s.td3.seq", slugify(arr.Title+"_"+arr.RootKey), sl.Slug)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	w.Write(data)
}

func (s *Server) handleExportSVG(w http.ResponseWriter, r *http.Request) {
	arr, err := s.resolveArrangement(r)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	sliceIdx, _ := strconv.Atoi(r.URL.Query().Get("slice"))
	svgStr := RenderArrangementSVG(arr, sliceIdx)
	fname := fmt.Sprintf("%s_%s_chart.svg", slugify(arr.Title), arr.RootKey)
	w.Header().Set("Content-Type", "image/svg+xml")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	}
	w.Write([]byte(svgStr))
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('_')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

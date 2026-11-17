package web

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>IMAGINATION ENGINE // Modular &amp; Hybrid Harmony Studio</title>
<style>
  :root {
    --bg: #070a12;
    --panel: #0e1422;
    --panel-alt: #131c31;
    --border: #1e293b;
    --text: #f8fafc;
    --muted: #94a3b8;
    --amber: #f59e0b;
    --cyan: #38bdf8;
    --purple: #a855f7;
    --coral: #fb7185;
    --emerald: #10b981;
  }
  * { box-sizing: border-box; }
  html, body {
    margin: 0; padding: 0;
    background: var(--bg);
    color: var(--text);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 13px;
    overflow-x: hidden;
  }
  header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 18px;
    background: #0a0f1d;
    border-bottom: 1px solid var(--border);
    flex-wrap: wrap;
    gap: 10px;
    position: sticky;
    top: 0;
    z-index: 100;
  }
  .brand {
    display: flex; align-items: center; gap: 12px;
  }
  .brand-badge {
    background: linear-gradient(135deg, var(--amber), var(--purple));
    color: #050811; font-weight: 900; padding: 4px 9px; border-radius: 5px;
    letter-spacing: 1px;
  }
  .transport-bar {
    display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
  }
  button, select, input[type="text"], input[type="number"] {
    background: var(--panel-alt);
    color: var(--text);
    border: 1px solid #334155;
    padding: 6px 10px;
    border-radius: 6px;
    font-family: inherit;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.12s ease;
  }
  button:hover, select:hover {
    border-color: var(--cyan);
    background: #1e293b;
  }
  button.primary {
    background: var(--amber);
    color: #050811;
    font-weight: 800;
    border-color: var(--amber);
  }
  button.primary:hover {
    background: #fbbf24;
  }
  button.playing {
    background: var(--coral);
    color: #050811;
    border-color: var(--coral);
    font-weight: 800;
  }
  button.active-tab {
    background: var(--cyan);
    color: #050811;
    font-weight: 800;
    border-color: var(--cyan);
  }
  .layout {
    display: grid;
    grid-template-columns: 320px minmax(0, 1fr);
    min-height: calc(100vh - 58px);
    align-items: start;
  }
  @media (max-width: 1100px) {
    .layout {
      grid-template-columns: 1fr;
    }
  }
  .sidebar {
    background: var(--panel);
    border-right: 1px solid var(--border);
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    position: sticky;
    top: 58px;
    max-height: calc(100vh - 58px);
    overflow-y: auto;
  }
  .main-pane {
    display: block;
    padding: 16px 20px 48px 20px;
    min-width: 0;
  }
  .main-pane > * {
    margin-bottom: 16px;
  }
  .section-box {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
  }
  .section-title {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--cyan);
    font-weight: 700;
    margin-bottom: 8px;
  }
  .prog-list {
    display: flex; flex-direction: column; gap: 6px;
    max-height: calc(100vh - 410px);
    min-height: 220px;
    overflow-y: auto;
    padding-right: 4px;
  }
  .prog-item {
    padding: 9px 10px;
    background: var(--panel-alt);
    border: 1px solid transparent;
    border-radius: 6px;
    cursor: pointer;
  }
  .prog-item:hover {
    border-color: #475569;
  }
  .prog-item.selected {
    border-color: var(--amber);
    background: #1c2438;
  }
  .prog-item-title {
    font-weight: 700; color: var(--text); margin-bottom: 3px;
  }
  .prog-item-meta {
    font-size: 11px; color: var(--muted);
  }
  .top-controls {
    display: flex; justify-content: space-between; align-items: center;
    flex-wrap: wrap; gap: 10px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 10px 14px;
  }
  .view-nav {
    display: flex; align-items: center; justify-content: space-between;
    flex-wrap: wrap; gap: 8px;
    background: #0b1324;
    border: 1px solid #26354d;
    border-radius: 8px;
    padding: 8px 12px;
  }
  .key-strip {
    display: flex; gap: 4px; align-items: center; flex-wrap: wrap;
  }
  .key-btn {
    padding: 4px 8px; font-size: 11px;
  }
  .key-btn.active {
    background: var(--purple);
    color: #fff;
    border-color: var(--purple);
    font-weight: 800;
  }
  .mixer-strip {
    display: grid;
    grid-template-columns: repeat(3, minmax(230px, 1fr));
    gap: 12px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
  }
  @media (max-width: 1050px) {
    .mixer-strip {
      grid-template-columns: 1fr;
    }
  }
  .mixer-card {
    background: var(--panel-alt);
    border: 1px solid #26334d;
    border-radius: 6px;
    padding: 10px 12px;
    display: flex;
    flex-direction: column;
    gap: 7px;
    min-width: 0;
  }
  .mixer-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    font-size: 11.5px;
  }
  .mixer-row select {
    padding: 4px 6px;
    font-size: 11px;
    flex: 1;
    min-width: 0;
  }
  .oct-stepper {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
  }
  .oct-stepper select {
    flex: 1;
    text-align: center;
  }
  .oct-btn {
    padding: 3px 8px;
    font-weight: 900;
    font-size: 12px;
    line-height: 1;
  }
  .chords-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
    gap: 10px;
  }
  .chord-card {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    position: relative;
  }
  .chord-card.active-chord {
    border-color: var(--amber);
    box-shadow: 0 0 15px rgba(245, 158, 11, 0.25);
  }
  .chord-sym {
    font-size: 20px; font-weight: 800; color: #fff;
  }
  .chord-roman {
    font-size: 12px; color: var(--amber); font-weight: 700;
    float: right;
  }
  .chord-func {
    font-size: 11px; color: var(--cyan); margin: 4px 0 8px 0;
  }
  .chord-voicing {
    font-size: 11px; color: #cbd5e1;
    background: #080d19; padding: 5px 7px; border-radius: 4px;
    margin-bottom: 6px;
  }
  .chord-color {
    font-size: 11px; color: #fde047; margin-bottom: 6px;
  }
  .chord-tip {
    font-size: 11px; color: var(--muted); line-height: 1.35;
  }
  .step-table-wrap {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px;
    overflow-x: auto;
  }
  .step-grid {
    display: grid;
    grid-template-columns: 95px repeat(32, minmax(28px, 1fr));
    gap: 3px;
    align-items: center;
    font-size: 10.5px;
    min-width: 980px;
  }
  .step-cell {
    background: var(--panel-alt);
    border: 1px solid #1e293b;
    border-radius: 4px;
    padding: 5px 2px;
    text-align: center;
  }
  .step-cell.playhead {
    outline: 2px solid #fff;
    background: #334155 !important;
  }
  .step-cell.role-anchor { border-bottom: 3px solid var(--amber); }
  .step-cell.role-color { border-bottom: 3px solid var(--purple); }
  .step-cell.role-approach { border-bottom: 3px solid var(--coral); }
  .step-cell.role-pedal { border-bottom: 3px solid var(--emerald); }
  .svg-container {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px;
    overflow-x: auto;
  }
  .svg-container svg {
    width: 100%;
    min-width: 900px;
    height: auto;
    display: block;
    border-radius: 6px;
  }
  .slider-row {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 6px; font-size: 11.5px;
  }
  input[type="range"] {
    width: 130px; accent-color: var(--amber);
  }
  .bpm-box {
    display: flex; align-items: center; gap: 6px;
    background: #131c31; border: 1px solid #334155;
    padding: 3px 9px; border-radius: 6px;
  }
  .bpm-box input[type="range"] {
    width: 85px; accent-color: var(--cyan);
  }
  .bpm-box input[type="number"] {
    width: 54px; padding: 3px 5px; text-align: center;
    font-weight: 800; color: var(--amber); background: #070a12;
  }
  .journey-bar {
    display: flex; align-items: center; justify-content: space-between;
    flex-wrap: wrap; gap: 8px;
    background: #0d1629;
    border: 1px solid #283a59;
    border-radius: 8px;
    padding: 8px 12px;
    margin-bottom: 10px;
  }
  .stage-pill {
    padding: 4px 8px;
    border-radius: 5px;
    font-size: 11px;
    background: #152038;
    border: 1px solid #2e4166;
    color: var(--muted);
    cursor: pointer;
  }
  .stage-pill.active-stage {
    background: var(--amber);
    color: #050811;
    border-color: var(--amber);
    font-weight: 800;
  }
  .mod-meter {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #080d19;
    border: 1px solid #26354d;
    padding: 3px 8px;
    border-radius: 5px;
    font-size: 11px;
  }
</style>
</head>
<body>

<header>
  <div class="brand">
    <span class="brand-badge">IMAGINE // SEQ</span>
    <div>
      <div style="font-weight:800;font-size:14px;">IMAGINATION ENGINE STUDIO</div>
      <div style="font-size:11px;color:var(--muted);">Multi-Port Hardware &amp; Soft-Synth Harmony Engine (Vital / Surge XT / Crave / East Beast / Edge)</div>
    </div>
  </div>

  <div class="transport-bar">
    <button id="playBtn" class="primary" onclick="togglePlay()">▶ Audition</button>

    <!-- Engine Density / Platform Profile Selector (v3 Mac Sketch Mode auto-selected on macOS!) -->
    <div class="bpm-box" title="v3 = Ultra-Sparse Mac Sketch Mode (1-2 notes per 4 beats) | v2 = Refined Sparse + MIDI CC#1/CC#74 Modulation | v1 = Legacy Dense">
      <span style="font-size:11px;color:var(--amber);font-weight:700;">Engine Mode:</span>
      <select id="profileSelect" onchange="onProfileChange()" style="padding:3px 6px;font-weight:700;color:#fff;">
        <option value="v3">v3: Mac Sketch Mode (Ultra-Sparse 1–2 Notes/Bar)</option>
        <option value="v2">v2: Refined Sparse + CC#1/CC#74 Mod</option>
        <option value="v1">v1: Legacy Dense (32-Step Arps)</option>
      </select>
    </div>

    <!-- Song Playback Mode: Full 9-Stage Song Journey vs Single Section Loop -->
    <div class="bpm-box" title="Play the entire 9-stage Song Journey automatically (Intro/Outro -> Build -> Modulated Drop -> Breakdown -> Ending) or loop 1 section">
      <span style="font-size:11px;color:var(--cyan);font-weight:700;">Song Mode:</span>
      <select id="songModeSelect" onchange="onSongModeChange()" style="padding:3px 6px;font-weight:700;">
        <option value="journey" selected>Full Song Journey (9-Stage Auto-Chain)</option>
        <option value="loop">Loop Selected Section Only</option>
      </select>
    </div>

    <div class="bpm-box" title="Defaults to 80–95 BPM so every note breathes clearly">
      <span style="font-size:11px;color:var(--cyan);font-weight:700;">BPM:</span>
      <input type="range" id="bpmSlider" min="40" max="200" step="1" value="86" oninput="onBpmChange(this.value)"/>
      <input type="number" id="bpmInput" min="40" max="200" step="1" value="86" onchange="onBpmChange(this.value)"/>
      <label style="font-size:10.5px;color:var(--muted);cursor:pointer;display:flex;align-items:center;gap:3px;" title="Lock current tempo when switching songs">
        <input type="checkbox" id="bpmLock"/> Lock
      </label>
    </div>

    <div style="display:flex;align-items:center;gap:4px;">
      <select id="midiOutSelect" onchange="selectMIDIOut()" title="Broadcast to ALL active synths (Vital + Surge XT + USB Hardware) simultaneously, or pick a single port">
        <option value="all">MIDI Out: ALL ACTIVE PORTS (Inboard + Outboard)</option>
      </select>
      <button onclick="rescanMIDIPorts()" title="Re-scan USB &amp; Virtual MIDI ports">↻</button>
    </div>

    <div class="bpm-box" title="Behringer Edge runs natively off channel-less MIDI Clock (0xF8 + Start/Stop) down the daisy chain—no Ch 3 note clutter needed!">
      <label style="font-size:11px;color:var(--emerald);font-weight:700;cursor:pointer;display:flex;align-items:center;gap:4px;">
        <input type="checkbox" id="midiClockToggle" checked onchange="syncBackendIfPlaying()"/> Clock (Edge Sync):
      </label>
      <select id="clockDivSelect" onchange="syncBackendIfPlaying()" style="padding:3px 6px;font-weight:700;">
        <option value="1" selected>1x (24 PPQN Standard)</option>
        <option value="2">1/2x (12 PPQN Half-Time)</option>
        <option value="4">1/4x (6 PPQN Beat Pulse)</option>
      </select>
    </div>
    <label style="font-size:11.5px;color:#e9d5ff;cursor:pointer;" title="Send MIDI CC#1 (Mod Wheel) &amp; CC#74 (Filter Cutoff) sweeps during Build-Up &amp; Drop">
      <input type="checkbox" id="midiModToggle" checked onchange="syncBackendIfPlaying()"/> CC#1/74 Mod
    </label>
    <label style="font-size:11.5px;color:var(--amber);cursor:pointer;" title="Fallback browser WebAudio preview (automatically OFF when MIDI synths are available to save CPU)">
      <input type="checkbox" id="browserSynthToggle"/> 🔊 Browser Preview
    </label>
  </div>
</header>

<div class="layout">
  <aside class="sidebar">
    <div class="section-box">
      <div class="section-title">1. Hybrid Imagination Generator</div>
      <div class="slider-row">
        <span>Primary Genre:</span>
        <select id="genPrimary" style="width:155px;">
          <option value="Trance">Trance / Berlin School</option>
          <option value="Drum & Bass">Drum &amp; Bass (Liquid)</option>
          <option value="Squarepusher IDM">Squarepusher / IDM</option>
          <option value="Esoteric Modular">Esoteric Modular</option>
          <option value="Lo-Fi">Lo-Fi / Jazz Guitar</option>
        </select>
      </div>
      <div class="slider-row">
        <span>Cross-Pollinate:</span>
        <select id="genSecondary" style="width:155px;">
          <option value="Squarepusher">Squarepusher / Acid-Jazz</option>
          <option value="Lo-Fi">Lo-Fi / Neo-Soul</option>
          <option value="Esoteric Modular">Esoteric Modular</option>
          <option value="Trance">Hypnotic Trance Pedal</option>
          <option value="Drum & Bass">Atmospheric Jungle</option>
        </select>
      </div>
      <div class="slider-row">
        <span>Color Density (9/#11):</span>
        <input type="range" id="colorSlider" min="0.1" max="1.0" step="0.05" value="0.70" oninput="fetchArrangement()"/>
      </div>
      <div class="slider-row">
        <span>Transition Complexity:</span>
        <input type="range" id="compSlider" min="0.1" max="1.0" step="0.05" value="0.60" oninput="fetchArrangement()"/>
      </div>
      <div style="display:flex;gap:6px;margin-top:8px;">
        <button class="primary" style="flex:1;" onclick="generateHybrid(false)">⚡ Synthesize</button>
        <button style="flex:1;" onclick="generateHybrid(true)">+ Save to DB</button>
      </div>
    </div>

    <div class="section-box" style="display:flex;flex-direction:column;">
      <div class="section-title">2. SQLite Progression Corpus (80–95 BPM)</div>
      <div style="display:flex;gap:6px;margin-bottom:8px;">
        <select id="genreFilter" onchange="loadProgressions()" style="flex:1;">
          <option value="all">All Progressions (Seed + Obsidian)</option>
          <option value="obsidian">Obsidian Clippings (music + chords)</option>
          <option value="trance">Trance / Berlin School</option>
          <option value="progressive">Progressive House &amp; Trance</option>
          <option value="deep house">Deep House / Dorian 7ths</option>
          <option value="drum">Drum &amp; Bass / Jungle</option>
          <option value="modular">Esoteric Modular</option>
          <option value="lofi">Lo-Fi / Neo-Soul</option>
          <option value="squarepusher">IDM / Acid-Jazz</option>
        </select>
      </div>
      <div id="progList" class="prog-list"></div>
    </div>
  </aside>

  <main class="main-pane">
    <div class="view-nav">
      <div style="display:flex;gap:6px;align-items:center;flex-wrap:wrap;">
        <span style="font-size:11px;color:var(--muted);font-weight:700;text-transform:uppercase;margin-right:4px;">Workspace View:</span>
        <button id="viewBtnAll" class="active-tab" onclick="setViewMode('all')">All Panels (Scrollable)</button>
        <button id="viewBtnChords" onclick="setViewMode('chords')">Chords &amp; Voice Mixer</button>
        <button id="viewBtnSeq" onclick="setViewMode('seq')">32-Step Hardware Sequencer</button>
        <button id="viewBtnChart" onclick="setViewMode('chart')">Vector Chart (.SVG / .PNG)</button>
      </div>
      <div style="display:flex;gap:6px;flex-wrap:wrap;">
        <button class="primary" onclick="downloadMIDI()">⬇ Full Song .MID (w/ CC Mod)</button>
        <button onclick="downloadCraveSeq('crave')">⬇ Crave .SEQ (32-Step)</button>
        <button onclick="downloadCraveSeq('eastbeast')">⬇ East Beast .SEQ</button>
        <button onclick="downloadTD3Seq()">⬇ TD-3 .SEQ</button>
        <button onclick="downloadSVG()">⬇ Chart .SVG</button>
        <button onclick="downloadPNG()">⬇ Chart .PNG</button>
      </div>
    </div>

    <div class="top-controls">
      <div>
        <div id="arrTitle" style="font-size:18px;font-weight:800;color:#fff;">Loading...</div>
        <div id="arrMeta" style="font-size:12px;color:var(--cyan);margin-top:2px;"></div>
      </div>
      <div class="key-strip">
        <span style="font-size:11.5px;color:var(--muted);margin-right:4px;">Transpose Key:</span>
        <div id="keyButtons" style="display:flex;gap:4px;flex-wrap:wrap;"></div>
      </div>
    </div>

    <!-- 4-Column Per-Part Port, Channel & Octave Mixer (Run Vital + Surge XT + Crave + East Beast + Edge together!) -->
    <div id="panelMixer" class="mixer-strip">
      <!-- 1. Poly Chords (2-3 Note Pad Shells) -->
      <div class="mixer-card" style="border-top:3px solid var(--purple);">
        <div class="mixer-row">
          <label style="font-weight:700;color:#e9d5ff;cursor:pointer;display:flex;align-items:center;gap:6px;">
            <input type="checkbox" id="vPoly" checked onchange="syncBackendIfPlaying()"/> Poly Pad (2–3 Notes Max)
          </label>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Port:</span>
          <select id="polyPort" class="voice-port-select" onchange="onVoiceConfigChange()">
            <option value="all">All Active Ports</option>
          </select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">MIDI Ch:</span>
          <select id="polyCh" onchange="onVoiceConfigChange()"></select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Octave:</span>
          <div class="oct-stepper">
            <button class="oct-btn" onclick="stepOctave('polyOct', -1)">−</button>
            <select id="polyOct" onchange="onVoiceConfigChange()">
              <option value="-2">-2 Oct</option>
              <option value="-1">-1 Oct</option>
              <option value="0" selected>0 (Mid Pad)</option>
              <option value="1">+1 Oct</option>
              <option value="2">+2 Oct</option>
            </select>
            <button class="oct-btn" onclick="stepOctave('polyOct', 1)">+</button>
          </div>
        </div>
      </div>

      <!-- 2. Behringer Crave (Defaults to -1 Oct Sub) -->
      <div class="mixer-card" style="border-top:3px solid var(--amber);">
        <div class="mixer-row">
          <label style="font-weight:700;color:var(--amber);cursor:pointer;display:flex;align-items:center;gap:6px;">
            <input type="checkbox" id="vCrave" checked onchange="syncBackendIfPlaying()"/> Behringer Crave (Sub)
          </label>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Port:</span>
          <select id="cravePort" class="voice-port-select" onchange="onVoiceConfigChange()">
            <option value="all">All Active Ports</option>
          </select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">MIDI Ch:</span>
          <select id="craveCh" onchange="onVoiceConfigChange()"></select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Octave:</span>
          <div class="oct-stepper">
            <button class="oct-btn" onclick="stepOctave('craveOct', -1)">−</button>
            <select id="craveOct" onchange="onVoiceConfigChange()">
              <option value="-2">-2 Oct (Deep Sub)</option>
              <option value="-1" selected>-1 Oct (Sub Default)</option>
              <option value="0">0 (Bass)</option>
              <option value="1">+1 Oct (Baritone)</option>
              <option value="2">+2 Oct (Lead)</option>
            </select>
            <button class="oct-btn" onclick="stepOctave('craveOct', 1)">+</button>
          </div>
        </div>
      </div>

      <!-- 3. East Beast (Defaults to +1 Oct High) -->
      <div class="mixer-card" style="border-top:3px solid var(--cyan);">
        <div class="mixer-row">
          <label style="font-weight:700;color:var(--cyan);cursor:pointer;display:flex;align-items:center;gap:6px;">
            <input type="checkbox" id="vEast" checked onchange="syncBackendIfPlaying()"/> East Beast (High Lead)
          </label>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Port:</span>
          <select id="eastPort" class="voice-port-select" onchange="onVoiceConfigChange()">
            <option value="all">All Active Ports</option>
          </select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">MIDI Ch:</span>
          <select id="eastCh" onchange="onVoiceConfigChange()"></select>
        </div>
        <div class="mixer-row">
          <span style="color:var(--muted);width:58px;">Octave:</span>
          <div class="oct-stepper">
            <button class="oct-btn" onclick="stepOctave('eastOct', -1)">−</button>
            <select id="eastOct" onchange="onVoiceConfigChange()">
              <option value="-2">-2 Oct (Low)</option>
              <option value="-1">-1 Oct (Warm)</option>
              <option value="0">0 (Midrange)</option>
              <option value="1" selected>+1 Oct (High Default)</option>
              <option value="2">+2 Oct</option>
            </select>
            <button class="oct-btn" onclick="stepOctave('eastOct', 1)">+</button>
          </div>
        </div>
      </div>
    </div>

    <div id="panelChords">
      <div id="theoryBanner" class="section-box" style="display:grid;grid-template-columns:1fr 1fr;gap:14px;margin-bottom:14px;">
        <div>
          <div class="section-title">Harmonic &amp; Transition Architecture</div>
          <div id="theoryNotes" style="color:#e2e8f0;font-size:12px;"></div>
        </div>
        <div>
          <div class="section-title">Hardware Patch &amp; Multi-Synth Routing Suggestion</div>
          <div id="patchNotes" style="color:#a7f3d0;font-size:12px;"></div>
        </div>
      </div>

      <div id="chordsContainer" class="chords-row"></div>
    </div>

    <div id="panelSeq" class="step-table-wrap">
      <!-- 9-Stage Full Song Journey Chain Bar + Live MIDI CC#1 & CC#74 Modulation Telemetry -->
      <div class="journey-bar">
        <div style="display:flex;align-items:center;gap:5px;flex-wrap:wrap;">
          <span style="font-size:11px;font-weight:800;color:var(--amber);text-transform:uppercase;margin-right:4px;">Song Journey Chain:</span>
          <div id="journeyChainStrip" style="display:flex;gap:4px;flex-wrap:wrap;"></div>
        </div>
        <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap;">
          <span class="mod-meter" title="Live MIDI CC#1 (Mod Wheel) sent to synths">
            <span style="color:var(--purple);font-weight:700;">CC#1 Mod:</span>
            <strong id="cc1Val" style="color:#fff;">12</strong>
          </span>
          <span class="mod-meter" title="Live MIDI CC#74 (Filter Cutoff) sent to synths">
            <span style="color:var(--cyan);font-weight:700;">CC#74 Cutoff:</span>
            <strong id="cc74Val" style="color:#fff;">48</strong>
          </span>
        </div>
      </div>

      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;flex-wrap:wrap;gap:8px;">
        <div style="display:flex;gap:6px;align-items:center;flex-wrap:wrap;">
          <span class="section-title" style="margin:0;margin-right:6px;">3. Song Section Pattern Slices (32 Steps):</span>
          <button id="sliceBtn0" class="active-tab" onclick="selectSlice(0)">1. Intro (Minimal)</button>
          <button id="sliceBtn1" onclick="selectSlice(1)">2. Build-Up (+Approach &amp; Mod)</button>
          <button id="sliceBtn2" onclick="selectSlice(2)">3. Drop (CC#1/74 Modulated)</button>
          <button id="sliceBtn3" onclick="selectSlice(3)">4. Breakdown (Spacious Pedal)</button>
          <button id="sliceBtn4" onclick="selectSlice(4)">5. Outro (Mellow Resolve)</button>
        </div>
        <div id="sliceDesc" style="font-size:11.5px;color:var(--amber);"></div>
      </div>
      <div id="stepGridContainer"></div>
    </div>

    <div id="panelChart" class="svg-container">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;flex-wrap:wrap;gap:8px;">
        <span class="section-title" style="margin:0;">4. Rendered Harmonic &amp; Sequencer Vector Chart (Live SVG / PNG Export)</span>
        <span style="font-size:11px;color:var(--muted);"> Amber = Anchor/Root | Purple = Color (9th/#11th) | Coral = Chromatic Transition/Turnaround | Emerald = Modal Pedal</span>
      </div>
      <div id="svgMount"></div>
    </div>
  </main>
</div>

<script>
const KEYS = ["C","C#","D","Eb","E","F","F#","G","Ab","A","Bb","B"];
let currentArr = null;
let currentSource = "db";
let currentID = 1;
let currentKey = "";
let userSelectedKey = "";
let currentSlice = 0;
let currentStage = 0;
let currentSeed = 42;
let currentBPM = 86;
let currentProfile = "v3";
let bpmUserModified = false;

let audioCtx = null;
let isPlaying = false;
let currentStep = 0;
let prevHighlightedStep = -1;
let prevHighlightedChord = -1;
let statusPollTimer = null;
let previewStepTimer = null;
let backendPorts = [];

async function init() {
  initChannelSelects();
  renderKeyButtons();
  await initMIDIPorts();
  await loadProgressions();
  await fetchArrangement();
}

function setViewMode(mode) {
  const btns = {
    all: "viewBtnAll",
    chords: "viewBtnChords",
    seq: "viewBtnSeq",
    chart: "viewBtnChart"
  };
  Object.keys(btns).forEach(k => {
    const el = document.getElementById(btns[k]);
    if (el) el.className = (k === mode) ? "active-tab" : "";
  });

  const pMixer = document.getElementById("panelMixer");
  const pChords = document.getElementById("panelChords");
  const pSeq = document.getElementById("panelSeq");
  const pChart = document.getElementById("panelChart");

  if (mode === "all") {
    pMixer.style.display = "grid";
    pChords.style.display = "block";
    pSeq.style.display = "block";
    pChart.style.display = "block";
  } else if (mode === "chords") {
    pMixer.style.display = "grid";
    pChords.style.display = "block";
    pSeq.style.display = "none";
    pChart.style.display = "none";
  } else if (mode === "seq") {
    pMixer.style.display = "grid";
    pChords.style.display = "none";
    pSeq.style.display = "block";
    pChart.style.display = "none";
  } else if (mode === "chart") {
    pMixer.style.display = "none";
    pChords.style.display = "none";
    pSeq.style.display = "none";
    pChart.style.display = "block";
  }
}

function stepOctave(selectId, delta) {
  const el = document.getElementById(selectId);
  if (!el) return;
  let val = parseInt(el.value, 10) + delta;
  if (val < -2) val = -2;
  if (val > 2) val = 2;
  el.value = String(val);
  onVoiceConfigChange();
}

function initChannelSelects() {
  const configs = [
    { id: "polyCh", def: 3, label0: "Ch 1 (Omni / Raw 0)" },
    { id: "craveCh", def: 0, label0: "Ch 1 (Crave Default)" },
    { id: "eastCh", def: 1, label0: "Ch 1 (Omni / Raw 0)" }
  ];
  configs.forEach(cfg => {
    const sel = document.getElementById(cfg.id);
    if (!sel) return;
    sel.innerHTML = "";
    for (let ch = 0; ch < 16; ch++) {
      const opt = document.createElement("option");
      opt.value = String(ch);
      opt.textContent = (ch === 0) ? cfg.label0 : ("Ch " + (ch + 1) + " (Raw " + ch + ")");
      if (ch === cfg.def) opt.selected = true;
      sel.appendChild(opt);
    }
  });
}

function getVoiceSettings() {
  return {
    poly_octave: parseInt(document.getElementById("polyOct").value, 10) || 0,
    crave_octave: parseInt(document.getElementById("craveOct").value, 10) || 0,
    east_beast_octave: parseInt(document.getElementById("eastOct").value, 10) || 0,
    edge_octave: 0,
    poly_channel: parseInt(document.getElementById("polyCh").value, 10) || 0,
    crave_channel: parseInt(document.getElementById("craveCh").value, 10) || 0,
    east_beast_channel: parseInt(document.getElementById("eastCh").value, 10) || 0,
    edge_channel: 2,
    poly_port_id: document.getElementById("polyPort").value || "all",
    crave_port_id: document.getElementById("cravePort").value || "all",
    east_beast_port_id: document.getElementById("eastPort").value || "all",
    edge_port_id: "none"
  };
}

async function onVoiceConfigChange() {
  await fetchArrangement();
  syncBackendIfPlaying();
}

async function onProfileChange() {
  currentProfile = document.getElementById("profileSelect").value;
  await fetchArrangement();
  syncBackendIfPlaying();
}

function onSongModeChange() {
  syncBackendIfPlaying();
}

async function initMIDIPorts() {
  await populateMIDISelects();
}

async function rescanMIDIPorts() {
  await populateMIDISelects();
  syncBackendIfPlaying();
}

async function populateMIDISelects() {
  const globalSel = document.getElementById("midiOutSelect");
  const prevGlobal = globalSel.value || "all";

  backendPorts = [];
  try {
    const res = await fetch("/api/midi/ports");
    const data = await res.json();
    if (data && data.default_profile && !window._profileInitialized) {
      currentProfile = data.default_profile;
      document.getElementById("profileSelect").value = currentProfile;
      window._profileInitialized = true;
    }
    if (data && Array.isArray(data.ports)) {
      backendPorts = data.ports;
    }
  } catch (e) {}

  globalSel.innerHTML = "";
  const optAll = document.createElement("option");
  optAll.value = "all";
  optAll.textContent = "MIDI Out: ALL ACTIVE PORTS (" + backendPorts.length + " Connected)";
  globalSel.appendChild(optAll);

  backendPorts.forEach(p => {
    const opt = document.createElement("option");
    opt.value = p.id;
    opt.textContent = "MIDI Out: " + p.name;
    globalSel.appendChild(opt);
  });

  const optNone = document.createElement("option");
  optNone.value = "none";
  optNone.textContent = "MIDI Out: Off (Browser Preview Only)";
  globalSel.appendChild(optNone);

  if (prevGlobal && Array.from(globalSel.options).some(o => o.value === prevGlobal)) {
    globalSel.value = prevGlobal;
  } else {
    globalSel.value = "all";
  }

  // Also populate per-voice Port dropdowns on the 3 mixer cards
  const voicePortIds = ["polyPort", "cravePort", "eastPort"];
  voicePortIds.forEach(id => {
    const sel = document.getElementById(id);
    if (!sel) return;
    const prev = sel.value || "all";
    sel.innerHTML = "";
    const oAll = document.createElement("option");
    oAll.value = "all";
    oAll.textContent = "All Active Ports (By Ch)";
    sel.appendChild(oAll);
    backendPorts.forEach(p => {
      const o = document.createElement("option");
      o.value = p.id;
      o.textContent = p.name;
      sel.appendChild(o);
    });
    const oMute = document.createElement("option");
    oMute.value = "none";
    oMute.textContent = "Muted on MIDI";
    sel.appendChild(oMute);
    if (Array.from(sel.options).some(o => o.value === prev)) {
      sel.value = prev;
    }
  });

  // Browser synth is OFF by default when MIDI ports exist (only auto-enable if zero MIDI ports exist)
  if (!window._browserAudioInitialized) {
    document.getElementById("browserSynthToggle").checked = (backendPorts.length === 0);
    window._browserAudioInitialized = true;
  }
}

function selectMIDIOut() {
  syncBackendIfPlaying();
}

function renderKeyButtons() {
  const wrap = document.getElementById("keyButtons");
  wrap.innerHTML = "";
  const origBtn = document.createElement("button");
  origBtn.className = "key-btn" + (userSelectedKey === "" ? " active" : "");
  origBtn.textContent = "Original";
  origBtn.title = "Use each song's original stored key";
  origBtn.onclick = async () => {
    userSelectedKey = "";
    currentKey = "";
    renderKeyButtons();
    await fetchArrangement();
    await loadProgressions();
    syncBackendIfPlaying();
  };
  wrap.appendChild(origBtn);

  KEYS.forEach(k => {
    const btn = document.createElement("button");
    const isActive = (userSelectedKey ? (k === userSelectedKey) : (k === currentKey));
    btn.className = "key-btn" + (isActive ? " active" : "");
    btn.textContent = k;
    btn.onclick = async () => {
      userSelectedKey = k;
      currentKey = k;
      renderKeyButtons();
      await fetchArrangement();
      await loadProgressions();
      syncBackendIfPlaying();
    };
    wrap.appendChild(btn);
  });
}

async function loadProgressions() {
  const genre = document.getElementById("genreFilter").value;
  const keyParam = userSelectedKey ? ("&key=" + encodeURIComponent(userSelectedKey)) : "";
  const res = await fetch("/api/progressions?genre=" + encodeURIComponent(genre) + keyParam, { cache: "no-store" });
  const list = await res.json();
  const wrap = document.getElementById("progList");
  wrap.innerHTML = "";
  list.forEach(item => {
    const div = document.createElement("div");
    div.id = "progItem_" + item.id;
    div.className = "prog-item" + (currentSource === "db" && item.id === currentID ? " selected" : "");
    const chordSyms = (item.chords || []).map(c => c.symbol).join(" → ");
    div.innerHTML = '<div class="prog-item-title">[#' + String(item.id).padStart(2, '0') + '] ' + item.title + '</div>' +
      '<div class="prog-item-meta">' + item.genre + ' | Key: ' + item.root_key + ' | ' + item.bpm + ' BPM</div>' +
      '<div style="font-size:11px;color:#38bdf8;margin-top:3px;">' + chordSyms + '</div>';
    div.onclick = async () => {
      currentSource = "db";
      currentID = item.id;
      currentKey = userSelectedKey || "";
      if (!document.getElementById("bpmLock").checked) {
        bpmUserModified = false;
      }
      await fetchArrangement();
      syncBackendIfPlaying();
      loadProgressions();
    };
    wrap.appendChild(div);
  });
}

function onBpmChange(val) {
  let v = parseInt(val, 10);
  if (isNaN(v) || v < 40) v = 40;
  if (v > 200) v = 200;
  currentBPM = v;
  bpmUserModified = true;
  document.getElementById("bpmSlider").value = v;
  document.getElementById("bpmInput").value = v;
  if (currentArr) {
    currentArr.bpm = v;
    document.getElementById("arrMeta").textContent =
      currentArr.genre + "  •  Key: " + currentArr.root_key + " (" + currentArr.mode + ")  •  " + currentBPM + " BPM  •  Profile: " + currentProfile.toUpperCase();
    syncBackendIfPlaying();
  }
}

function buildQueryParams() {
  const comp = document.getElementById("compSlider").value;
  const col = document.getElementById("colorSlider").value;
  const vs = getVoiceSettings();
  const p = new URLSearchParams({
    source: currentSource,
    id: currentID,
    key: userSelectedKey || currentKey,
    profile: currentProfile,
    complexity: comp,
    color: col,
    seed: currentSeed,
    slice: currentSlice,
    primary: document.getElementById("genPrimary").value,
    secondary: document.getElementById("genSecondary").value,
    crave_oct: vs.crave_octave,
    east_oct: vs.east_beast_octave,
    edge_oct: vs.edge_octave,
    poly_oct: vs.poly_octave,
    crave_ch: vs.crave_channel,
    east_ch: vs.east_beast_channel,
    edge_ch: vs.edge_channel,
    poly_ch: vs.poly_channel,
    poly_port: vs.poly_port_id,
    crave_port: vs.crave_port_id,
    east_port: vs.east_beast_port_id,
    edge_port: vs.edge_port_id
  });
  if (bpmUserModified || document.getElementById("bpmLock").checked) {
    p.set("bpm", String(currentBPM));
  }
  return p.toString();
}

async function fetchArrangement() {
  const qs = buildQueryParams();
  const res = await fetch("/api/progression?" + qs, { cache: "no-store" });
  currentArr = await res.json();
  currentKey = currentArr.root_key;
  if (!bpmUserModified && !document.getElementById("bpmLock").checked && currentArr.bpm) {
    currentBPM = currentArr.bpm;
    document.getElementById("bpmSlider").value = currentBPM;
    document.getElementById("bpmInput").value = currentBPM;
  } else {
    currentArr.bpm = currentBPM;
  }
  renderKeyButtons();
  renderJourneyChain();
  renderArrangement();
  await refreshSVG();
}

async function generateHybrid(saveToDB) {
  currentSource = "hybrid";
  currentSeed = Math.floor(Math.random() * 900000) + 1000;
  const qs = buildQueryParams() + (saveToDB ? "&save=1" : "");
  const res = await fetch("/api/generate?" + qs);
  currentArr = await res.json();
  if (saveToDB && currentArr.id) {
    currentSource = "db";
    currentID = currentArr.id;
    await loadProgressions();
  }
  currentKey = currentArr.root_key;
  if (!document.getElementById("bpmLock").checked && currentArr.bpm) {
    currentBPM = currentArr.bpm;
    bpmUserModified = false;
    document.getElementById("bpmSlider").value = currentBPM;
    document.getElementById("bpmInput").value = currentBPM;
  } else {
    currentArr.bpm = currentBPM;
  }
  renderKeyButtons();
  renderJourneyChain();
  renderArrangement();
  await refreshSVG();
  syncBackendIfPlaying();
}

function renderJourneyChain() {
  const wrap = document.getElementById("journeyChainStrip");
  if (!wrap || !currentArr) return;
  const chain = currentArr.song_chain || [];
  wrap.innerHTML = "";
  chain.forEach((st, idx) => {
    const pill = document.createElement("span");
    pill.id = "stagePill" + idx;
    pill.className = "stage-pill" + (idx === currentStage ? " active-stage" : "");
    pill.textContent = st.label;
    pill.onclick = () => {
      currentStage = idx;
      selectSlice(st.slice_index);
      renderJourneyChain();
    };
    wrap.appendChild(pill);
  });
}

function selectSlice(idx) {
  currentSlice = idx;
  [0,1,2,3,4].forEach(i => {
    const el = document.getElementById("sliceBtn" + i);
    if (el) el.className = (i === idx) ? "active-tab" : "";
  });
  renderArrangement();
  refreshSVG();
  syncBackendIfPlaying();
}

function renderArrangement() {
  if (!currentArr) return;
  const allChordSyms = (currentArr.chords || []).map(c => c.symbol).join(" → ");
  document.getElementById("arrTitle").textContent = currentArr.title;
  document.getElementById("arrMeta").textContent =
    currentArr.genre + "  •  Key: " + currentArr.root_key + " (" + currentArr.mode + ")  •  Chords: " + allChordSyms + "  •  " + currentArr.bpm + " BPM  •  Profile: " + currentProfile.toUpperCase();
  document.getElementById("theoryNotes").textContent = currentArr.theory_notes;
  document.getElementById("patchNotes").textContent = currentArr.patch_suggestion;

  if (currentSource === "db" && currentArr.id) {
    const activeItem = document.getElementById("progItem_" + currentArr.id);
    if (activeItem) {
      activeItem.innerHTML = '<div class="prog-item-title">[#' + String(currentArr.id).padStart(2, '0') + '] ' + currentArr.title + '</div>' +
        '<div class="prog-item-meta">' + currentArr.genre + ' | Key: ' + currentArr.root_key + ' | ' + currentArr.bpm + ' BPM</div>' +
        '<div style="font-size:11px;color:#38bdf8;margin-top:3px;">' + allChordSyms + '</div>';
    }
  }

  const chordsWrap = document.getElementById("chordsContainer");
  chordsWrap.innerHTML = "";
  (currentArr.chords || []).forEach((ch, idx) => {
    const card = document.createElement("div");
    card.className = "chord-card";
    card.id = "chordCard" + idx;
    const vNotes = (ch.voicing_midi || []).map(midiToName).join(" · ");
    const cDesc = (ch.color_descriptions || []).join(" + ");
    card.innerHTML =
      '<span class="chord-roman">' + ch.roman + '</span>' +
      '<div class="chord-sym">' + ch.symbol + '</div>' +
      '<div class="chord-func">' + ch.function + '</div>' +
      '<div class="chord-voicing">Pad Shell (' + (ch.voicing_midi || []).length + ' notes): <strong>' + vNotes + '</strong></div>' +
      '<div class="chord-color">★ ' + cDesc + '</div>' +
      '<div class="chord-tip">→ ' + ch.transition_tip + '</div>';
    chordsWrap.appendChild(card);
  });

  const sl = (currentArr.slices || [])[currentSlice];
  if (!sl) return;
  document.getElementById("sliceDesc").textContent = sl.description;

  const gridWrap = document.getElementById("stepGridContainer");
  let html = '<div class="step-grid">';
  html += '<div style="font-weight:700;color:var(--muted);">STEP (1..32)</div>';
  for (let i = 1; i <= 32; i++) {
    html += '<div style="text-align:center;color:var(--muted);font-size:10px;">' + String(i).padStart(2, '0') + '</div>';
  }

  html += '<div style="font-weight:700;color:var(--amber);">CRAVE (SUB)</div>';
  sl.crave_steps.forEach((st, i) => {
    const cls = classifyRoleClass(st.role);
    const flags = (st.accent ? "A" : "") + (st.glide ? "G" : "");
    html += '<div id="craveStep'+i+'" class="step-cell '+cls+'" title="Step '+st.step+' ('+st.chord_sym+'): '+st.role+' | CC#1='+st.mod_cc1+' CC#74='+st.cutoff_cc74+'">' +
      '<div style="font-weight:700;color:' + (st.rest ? '#475569' : '#fff') + ';">' + (st.rest ? "·" : st.note_name) + '</div>' +
      '<div style="font-size:9px;color:#fde047;">' + ( flags || "&nbsp;" ) + '</div>' +
      '</div>';
  });

  html += '<div style="font-weight:700;color:var(--cyan);">EAST BEAST</div>';
  sl.east_beast_steps.forEach((st, i) => {
    const cls = classifyRoleClass(st.role);
    const flags = (st.accent ? "A" : "") + (st.glide ? "G" : "");
    html += '<div id="eastStep'+i+'" class="step-cell '+cls+'" title="Step '+st.step+' ('+st.chord_sym+'): '+st.role+'">' +
      '<div style="font-weight:700;color:' + (st.rest ? '#475569' : '#38bdf8') + ';">' + (st.rest ? "·" : st.note_name) + '</div>' +
      '<div style="font-size:9px;color:#fde047;">' + ( flags || "&nbsp;" ) + '</div>' +
      '</div>';
  });

  html += '</div>';
  gridWrap.innerHTML = html;
  prevHighlightedStep = -1;
}

function classifyRoleClass(role) {
  const r = (role || "").toLowerCase();
  if (r.includes("approach") || r.includes("turnaround") || r.includes("lead ->")) return "role-approach";
  if (r.includes("color") || r.includes("9th") || r.includes("tension") || r.includes("mod")) return "role-color";
  if (r.includes("pedal")) return "role-pedal";
  return "role-anchor";
}

async function refreshSVG() {
  const res = await fetch("/api/export/svg?" + buildQueryParams());
  const svgText = await res.text();
  document.getElementById("svgMount").innerHTML = svgText;
}

function midiToName(m) {
  const names = ["C","C#","D","D#","E","F","F#","G","G#","A","A#","B"];
  return names[m % 12] + (Math.floor(m / 12) - 1);
}

function midiToFreq(m) {
  return 440 * Math.pow(2, (m - 69) / 12);
}

function syncBackendIfPlaying() {
  if (!isPlaying) return;
  const portID = document.getElementById("midiOutSelect").value || "all";
  fetch("/api/transport/sync", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      playing: true,
      port_id: portID,
      song_mode: document.getElementById("songModeSelect").value,
      slice_idx: currentSlice,
      stage_idx: currentStage,
      bpm: currentBPM || 86,
      send_clock: document.getElementById("midiClockToggle").checked,
      clock_div: parseInt(document.getElementById("clockDivSelect").value, 10) || 1,
      send_mod_cc: document.getElementById("midiModToggle").checked,
      enable_poly: document.getElementById("vPoly").checked,
      enable_crave: document.getElementById("vCrave").checked,
      enable_east: document.getElementById("vEast").checked,
      enable_edge: false,
      voice_config: getVoiceSettings(),
      arrangement: currentArr
    })
  }).catch(() => {});
}

function togglePlay() {
  const btn = document.getElementById("playBtn");
  if (isPlaying) {
    isPlaying = false;
    if (statusPollTimer) { clearInterval(statusPollTimer); statusPollTimer = null; }
    if (previewStepTimer) { clearTimeout(previewStepTimer); previewStepTimer = null; }
    btn.textContent = "▶ Audition";
    btn.classList.remove("playing");
    fetch("/api/transport/stop", { method: "POST" }).catch(() => {});
    return;
  }

  isPlaying = true;
  currentStep = 0;
  btn.textContent = "⏹ Stop Sequencer";
  btn.classList.add("playing");

  syncBackendIfPlaying();

  // Lightweight status poll (~5.5 Hz) so browser CPU stays under 2% on Mac Core i5!
  statusPollTimer = setInterval(pollBackendStatus, 180);

  if (document.getElementById("browserSynthToggle").checked) {
    if (!audioCtx) {
      audioCtx = new (window.AudioContext || window.webkitAudioContext)();
    }
    if (audioCtx.state === "suspended") audioCtx.resume();
    scheduleBrowserPreviewStep();
  }
}

async function pollBackendStatus() {
  if (!isPlaying || !currentArr) return;
  try {
    const res = await fetch("/api/transport/status");
    const st = await res.json();
    if (!st.running) return;

    document.getElementById("cc1Val").textContent = st.mod_cc1;
    document.getElementById("cc74Val").textContent = st.cutoff_cc74;

    if (st.stage_idx !== currentStage && document.getElementById("songModeSelect").value === "journey") {
      currentStage = st.stage_idx;
      renderJourneyChain();
    }
    if (st.slice_idx !== currentSlice) {
      currentSlice = st.slice_idx;
      [0,1,2,3,4].forEach(i => {
        const el = document.getElementById("sliceBtn" + i);
        if (el) el.className = (i === currentSlice) ? "active-tab" : "";
      });
      renderArrangement();
    }

    updateStepHighlight(st.step);
  } catch (e) {}
}

function updateStepHighlight(step) {
  if (prevHighlightedStep === step) return;
  if (prevHighlightedStep >= 0) {
    ["craveStep", "eastStep"].forEach(prefix => {
      const oldEl = document.getElementById(prefix + prevHighlightedStep);
      if (oldEl) oldEl.classList.remove("playhead");
    });
  }
  ["craveStep", "eastStep"].forEach(prefix => {
    const newEl = document.getElementById(prefix + step);
    if (newEl) newEl.classList.add("playhead");
  });
  prevHighlightedStep = step;

  if (currentArr && currentArr.chords && currentArr.chords.length > 0) {
    const numChords = currentArr.chords.length;
    const stepsPerChord = Math.floor(32 / numChords);
    const activeChordIdx = Math.floor(step / stepsPerChord) % numChords;
    if (activeChordIdx !== prevHighlightedChord) {
      const oldCard = document.getElementById("chordCard" + prevHighlightedChord);
      if (oldCard) oldCard.classList.remove("active-chord");
      const newCard = document.getElementById("chordCard" + activeChordIdx);
      if (newCard) newCard.classList.add("active-chord");
      prevHighlightedChord = activeChordIdx;
    }
  }
}

// Optional lightweight WebAudio preview ONLY when Browser Preview checkbox is explicitly enabled
function scheduleBrowserPreviewStep() {
  if (!isPlaying || !currentArr || !document.getElementById("browserSynthToggle").checked) return;
  const sl = currentArr.slices[currentSlice];
  const bpm = currentBPM || currentArr.bpm || 86;
  const stepDurSec = (60 / bpm) / 4;

  if (!audioCtx) {
    audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  }
  const now = audioCtx.currentTime;
  const numChords = currentArr.chords.length;
  const stepsPerChord = Math.floor(32 / numChords);
  const activeChordIdx = Math.floor(currentStep / stepsPerChord) % numChords;

  if (currentStep % stepsPerChord === 0 && document.getElementById("vPoly").checked) {
    const ch = currentArr.chords[activeChordIdx];
    (ch.voicing_midi || []).forEach(note => {
      playSimpleTone(note, now, stepDurSec * stepsPerChord * 0.9, "triangle", 0.04);
    });
  }

  const cStep = sl.crave_steps[currentStep];
  if (cStep && !cStep.rest && document.getElementById("vCrave").checked) {
    playSimpleTone(cStep.midi_note, now, stepDurSec * 0.75, "sawtooth", 0.08);
  }

  const ebStep = sl.east_beast_steps[currentStep];
  if (ebStep && !ebStep.rest && document.getElementById("vEast").checked) {
    playSimpleTone(ebStep.midi_note, now, stepDurSec * 0.75, "triangle", 0.06);
  }

  currentStep = (currentStep + 1) % 32;
  previewStepTimer = setTimeout(scheduleBrowserPreviewStep, stepDurSec * 1000);
}

function playSimpleTone(midi, startTime, dur, type, peakGain) {
  const osc = audioCtx.createOscillator();
  const gain = audioCtx.createGain();
  osc.type = type;
  osc.frequency.setValueAtTime(midiToFreq(midi), startTime);
  gain.gain.setValueAtTime(0.0001, startTime);
  gain.gain.linearRampToValueAtTime(peakGain, startTime + 0.015);
  gain.gain.exponentialRampToValueAtTime(0.0001, startTime + Math.max(0.04, dur));
  osc.connect(gain);
  gain.connect(audioCtx.destination);
  osc.start(startTime);
  osc.stop(startTime + Math.max(0.05, dur + 0.01));
}

function downloadMIDI() {
  window.location.href = "/api/export/midi?" + buildQueryParams();
}

function downloadCraveSeq(voice) {
  window.location.href = "/api/export/crave-seq?" + buildQueryParams() + "&voice=" + encodeURIComponent(voice);
}

function downloadTD3Seq() {
  window.location.href = "/api/export/td3-seq?" + buildQueryParams();
}

function downloadSVG() {
  window.location.href = "/api/export/svg?" + buildQueryParams() + "&download=1";
}

function downloadPNG() {
  const svgEl = document.querySelector("#svgMount svg");
  if (!svgEl) return;
  const serializer = new XMLSerializer();
  const svgStr = serializer.serializeToString(svgEl);
  const svgBlob = new Blob([svgStr], { type: "image/svg+xml;charset=utf-8" });
  const url = URL.createObjectURL(svgBlob);
  const img = new Image();
  img.onload = () => {
    const canvas = document.createElement("canvas");
    canvas.width = 1260;
    canvas.height = 680;
    const ctx = canvas.getContext("2d");
    ctx.fillStyle = "#080b12";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    URL.revokeObjectURL(url);
    canvas.toBlob(blob => {
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = (currentArr ? currentArr.title.replace(/\s+/g, "_") : "chart") + ".png";
      a.click();
    }, "image/png");
  };
  img.src = url;
}

window.addEventListener("DOMContentLoaded", init);
</script>
</body>
</html>
`

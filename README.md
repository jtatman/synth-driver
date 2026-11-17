# Imagination Engine (`imagine`)

A Go terminal engine and interactive localhost studio (`http://127.0.0.1:8090`) designed to break out of the **"8-bar major/minor modular loop trap"**.

It bridges full-song polyphonic harmonic progressions (with 9ths, #11ths, altered dominants, modal pedal points, and chromatic/tritone transitions) and hardware/soft-synth sequences across a complete **5-Part Song Structure** for:
- **Vital Soft-Synth / Polyphonic DAW Pads** (Default MIDI **Channel 1 / Raw Channel 0 (Omni)** so Linux/macOS Vital receives full polyphonic voice-led chords out of the box, with customizable channel `1..16` and octave shift `-2..+2`)
- **Behringer Crave** (Default MIDI **Channel 1** + native 32-step `.crave.seq` pattern slices with Glide, Accent, Gate length `0..7`, Ratchets, and independent Octave shift `-2..+2`)
- **Cre8audio East Beast** (Default MIDI **Channel 2**, tuned to a warm midrange register `G2..E4` so it is never chirpy, plus independent Octave shift `-2..+2` and `.seq` export)
- **Behringer Edge** (Default MIDI **Channel 3**, 8-step polymetric percussion & pitch accents, with independent Octave shift `-2..+2`)

---

## Key Features & Recent Upgrades

### 1. Native Go Backend MIDI Sequencer (Zero Browser Lag in Firefox & Safari)
- Instead of relying on browser WebMIDI (which Firefox and Safari restrict or stutter on at high BPMs), the Go binary itself binds directly to:
  - **Linux ALSA Sequencer (`libasound.so.2` + `/proc/asound/seq/clients`)**: Automatically discovers running soft-synths like **Vital (`seq:128:0`)**, **Midi Through (`seq:14:0`)**, and USB-MIDI synths—even in Firefox!
  - **macOS CoreMIDI (`CoreMIDI.framework`)**: Automatically discovers physical USB-MIDI synths like **Behringer CRAVE** (`coremidi:0`) AND creates a **Virtual CoreMIDI Source (`ImaginationEngine Virtual Out`)** that you can select inside Vital, Logic, or GarageBand on macOS!
- All hardware/soft-synth MIDI clocking (`24 PPQN` `0xF8` pulses) and note triggering runs inside a high-precision native Go goroutine (`pkg/midiout/engine.go`), eliminating Firefox/Safari audio/network stutter.
- Includes a **`🔊 Browser Synth` master toggle** in the top bar so you can completely disable the browser's built-in WebAudio synth when routing to Vital or hardware synths.

### 2. Per-Part MIDI Channel (`Ch 1 / Omni` .. `Ch 16`) & Octave Range (`-2` .. `+2` Oct) Mixer Strip
- Every voice (**Poly Chords / Vital**, **Crave**, **East Beast**, **Edge**) has its own card in the Mixer Strip with:
  - **Enable / Mute checkbox** (to isolate just Poly Chords into Vital on Omni Ch 1, or combine any layers)
  - **MIDI Channel selector (`Ch 1 (Omni/Vital)` through `Ch 16`)**
  - **Octave Transpose (`-2 Oct`, `-1 Oct`, `0`, `+1 Oct`, `+2 Oct`)** — applies in real time to live MIDI output, WebAudio audition, exported `.mid` files, `.seq` files, and `.svg`/`.png` charts.
- The **East Beast** default register has been lowered into the warm midrange (`G2..E4`, MIDI `43..64`) so lead/counter-harmony lines sit musically above the Crave bass without piercing high frequencies.

### 3. 5-Part Authentic Song Structure (Replacing the 3-Slice Break Layout)
Every progression is automatically arranged into **5 distinct 32-step song sections**:
1. **`1. Intro (Minimal Pulse & Bass)`** (`part_1_intro`) — Edge runs the rhythmic pulse while Crave plays a sparse root-fifth bassline; East Beast whispers only on bar downbeats.
2. **`2. Build-Up (Driving Groove & Approach)`** (`part_2_buildup`) — Crave locks into a 16th-note groove with 2-step chromatic approach glides; East Beast introduces midrange 3rd/7th guide tones.
3. **`3. Drop (Full Anthem & Midrange Arp)`** (`part_3_drop`) — Rolling offbeat root-octave Crave bass paired with an uplifting midrange East Beast arpeggio across 3rds, 7ths, and 9ths.
4. **`4. Breakdown (Lush Color & Modal Pedal)`** (`part_4_breakdown`) — Percussion drops back while Crave and East Beast sustain long, legato (`Gate 6–7`) 9th/#11th intervals around a shared modal pedal point.
5. **`5. Outro (Mellow Resolve & Wind-Down)`** (`part_5_outro`) — Strips away upper tensions, lengthens gate breathing space, and settles onto the tonic and fifth.

### 4. Corpus: 65 Progressions (8 Curated Seeds + 57 Obsidian Clippings)
All 7 markdown notes in `/home/james/brain/inbox/` tagged with `music` and `chords` are integrated into `imagination.db` and embedded into `pkg/db/obsidian_seed.json` via `//go:embed` for standalone macOS and Linux builds.

---

## Quick Start & CLI Reference

```bash
cd /home/james/.gemini/antigravity/scratch/imagination-engine

# 1. Start the interactive Web Studio (open http://127.0.0.1:8090 in Firefox, Safari, or Chrome)
./imagine serve --port 8090

# 2. Check detected native OS MIDI ports (Vital ALSA Sequencer on Linux / CoreMIDI on macOS)
./imagine midi-ports

# 3. List progressions in SQLite
./imagine list
./imagine list --genre obsidian

# 4. Export a full 5-part song bundle (.mid + 5 Crave .seq + 5 EastBeast .seq + 5 TD-3 .seq + .svg chart)
./imagine export --id 52 --key "F#" --bpm 138 --east-oct -1 --out ./output

# 5. Re-sync Obsidian vault notes tagged [music, chords] and rebuild Linux + macOS binaries
make sync-obsidian && make build-mac
```

### Pre-Compiled macOS Binaries (`dist/`)
- `dist/imagine-darwin-amd64` — Standalone binary for your **x86_64 Mac Mini** (includes native CoreMIDI + Virtual MIDI out + all 65 embedded progressions)
- `dist/imagine-darwin-arm64` — Standalone binary for **Apple Silicon Macs**

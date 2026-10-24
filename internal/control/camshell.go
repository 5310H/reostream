package control

import (
	"context"
	"sync"
	"time"

	"github.com/VoltTech21/reostream/internal/baichuan"
	"github.com/VoltTech21/reostream/internal/stream"
)

// The camera shell is what every page about one camera shares, per the
// redesign: a header with the camera switcher (dropdown, previous / next,
// and the [ ] keys), its state, model, address and firmware, the Live and
// Settings tabs, and on the Settings tab the category list down the left.
// The categories and their order are the design's CATMETA; one is listed
// only when this camera has it and reostream has a working page for it.

type camNavItem struct {
	ID, Label, Href string
	Current         bool
}

type camNavGroup struct {
	Title string
	Items []camNavItem
}

// camHead is the shell's data. Every camera page carries one as Head.
type camHead struct {
	Name     string
	Address  string
	Tab      string // "live" or "settings"
	Cat      string
	Prev     string
	Next     string
	Cameras  []railEntry
	State    string
	Model    string
	Firmware string
	// HasSD is a card in the camera; HasClips shows the Recordings tab
	// (a card, or clips the camera saved here).
	HasSD    bool
	HasClips bool
	Nav      []camNavGroup
	// CatLabel is the current category's name, for the page title.
	CatLabel string
}

// camIdent is what the header says about a camera that only the camera
// knows. It changes when firmware does, so it is kept for a while rather
// than read again on every page.
type camIdent struct {
	Support *baichuan.Support
	// HasSD is the camera reporting a storage card (its HddInfoList is not
	// empty).
	HasSD    bool
	Model    string
	Firmware string
	// Zoom is a second lens (TrackMix), over Baichuan or the camera's RTSP.
	Zoom bool
	at   time.Time
}

const identTTL = 10 * time.Minute

// skipIdent turns the header's own camera read off. Tests set it: their
// fake cameras answer a scripted sequence on one connection, and an extra
// dial for the header would take a reply meant for the page under test.
var skipIdent bool

var (
	identMu    sync.Mutex
	identCache = map[string]camIdent{}
)

// identFor returns the camera's support block, marketing model and
// firmware, from memory when it was read recently. A camera that cannot be
// reached gives a zero ident (nothing cached), and the shell shows less.
func (s *Server) identFor(ctx context.Context, cam Camera) camIdent {
	if skipIdent {
		return camIdent{}
	}
	key := cam.Name + "@" + cam.Address
	identMu.Lock()
	id, ok := identCache[key]
	identMu.Unlock()
	if ok && time.Since(id.at) < identTTL {
		return id
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	conn, err := s.dial(ctx, cam)
	if err != nil {
		return camIdent{}
	}
	defer conn.Close()
	id = camIdent{at: time.Now()}
	if sup, err := baichuan.GetSupport(ctx, conn); err == nil {
		id.Support = &sup
	}
	// versioninfo (80) carries the marketing model ("Reolink TrackMix PoE")
	// and the firmware version in one read. DeviceInfo's firmVersion is the
	// serial number on the TrackMix, so it is not used.
	if msg, ok := baichuan.ConfigMessages["versioninfo"]; ok {
		rctx, rcancel := context.WithTimeout(ctx, readTimeout)
		doc, status, err := baichuan.ReadConfig(rctx, conn, msg)
		rcancel()
		if err == nil && status == 200 {
			id.Model, _ = fieldValue(doc, "VersionInfo/type")
			id.Firmware, _ = fieldValue(doc, "VersionInfo/firmwareVersion")
		}
	}
	// A card in the slot: the support block's record (3 on an Argus with a
	// card, 0 on a TrackMix without one) or diskNum, which reads 0 for a
	// moment after a battery camera wakes. The sdcard block is no help: a
	// battery camera does not answer it at all.
	id.HasSD = id.Support != nil && (id.Support.Support.DiskNum > 0 || id.Support.Support.Record > 0)
	if cam.UID == "" && cam.Channel == 0 {
		zctx, zcancel := context.WithTimeout(ctx, 8*time.Second)
		_, _, _, zerr := stream.ZoomInfo(zctx, cam.Address, cam.Username, cam.Password)
		zcancel()
		id.Zoom = zerr == nil
	}
	identMu.Lock()
	identCache[key] = id
	identMu.Unlock()
	return id
}

// settingsCats is the design's CATMETA, cut to what exists. Each entry is
// shown only when its test passes for this camera.
func settingsCats(sup *baichuan.Support) []struct {
	group string
	items [][2]string
} {
	video := [][2]string{{"streams", "Encoding"}}
	if sup != nil && sup.HasFishEye(0) {
		video = append(video, [2]string{"view", "Fisheye view"})
	}
	if sup != nil && sup.HasStitch(0) {
		video = append(video, [2]string{"stitch", "Lens stitching"})
	}
	hardware := [][2]string{{"lights", "Lights"}, {"audio", "Audio and talk"}}
	if sup != nil && sup.HasPTZ() {
		hardware = append(hardware, [2]string{"ptz", "Pan, tilt, zoom"})
	}
	return []struct {
		group string
		items [][2]string
	}{
		{"Picture", [][2]string{{"image", "Image"}, {"overlay", "Name and timestamp"}}},
		{"Video", video},
		{"Hardware", hardware},
		{"Alerts", [][2]string{{"clips", "Auto-save motion clips"}, {"ftp", "FTP upload"}, {"email", "Email alerts"}, {"push", "Push notifications"}}},
		{"System", [][2]string{{"network", "Network services"}, {"time", "Date and time"}, {"remove", "Remove camera"}}},
		{"Expert", [][2]string{{"raw", "Raw values"}, {"advanced", "Raw documents"}}},
	}
}

// catHref is where a category lives. Most are sections of the settings
// tab; time, accounts and the raw documents keep their own pages and
// handlers, drawn inside the same shell.
func catHref(cam, id string) string {
	base := "/cameras/" + cam
	switch id {
	case "time", "accounts", "advanced", "raw":
		return base + "/" + id
	}
	return base + "/settings/" + id
}

// camHeadFor builds the shell for one camera page.
func (s *Server) camHeadFor(ctx context.Context, cam Camera, tab, cat string) camHead {
	h := camHead{Name: cam.Name, Address: cam.Address, Tab: tab, Cat: cat}
	if cams, err := s.fleet(); err == nil {
		h.Cameras = railEntries(cams, s.opts.Status)
		for i, c := range h.Cameras {
			if c.Name != cam.Name {
				continue
			}
			h.State = c.State
			if len(h.Cameras) > 1 {
				h.Prev = h.Cameras[(i+len(h.Cameras)-1)%len(h.Cameras)].Name
				h.Next = h.Cameras[(i+1)%len(h.Cameras)].Name
			}
		}
	}
	id := s.identFor(ctx, cam)
	// A battery camera records to its card by design, and right after it
	// wakes its support block can read as having none.
	h.Model, h.Firmware, h.HasSD = id.Model, id.Firmware, id.HasSD || cam.UID != ""
	h.HasClips = h.HasSD || s.hasSavedClips(cam.Name)
	for _, g := range settingsCats(id.Support) {
		ng := camNavGroup{Title: g.group}
		for _, it := range g.items {
			cur := it[0] == cat
			if cur {
				h.CatLabel = it[1]
			}
			ng.Items = append(ng.Items, camNavItem{ID: it[0], Label: it[1], Href: catHref(cam.Name, it[0]), Current: cur})
		}
		h.Nav = append(h.Nav, ng)
	}
	return h
}

// SwitchHref is where the camera switcher sends a camera: the same tab and
// category on the other camera.
func (h camHead) SwitchHref(name string) string {
	switch h.Tab {
	case "live":
		return "/cameras/" + name
	case "recordings":
		return "/cameras/" + name + "/recordings"
	}
	return catHref(name, h.Cat)
}

package ui

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// ImageButton is a simple clickable image (no background) that behaves like a button.
type ImageButton struct {
	widget.BaseWidget
	res   fyne.Resource
	img   *canvas.Image
	onTap func()
	size  fyne.Size
	hover bool
	mu    sync.RWMutex
}

var _ desktop.Hoverable = (*ImageButton)(nil)

func NewImageButton(res fyne.Resource, onTap func()) *ImageButton {
	ib := &ImageButton{res: res, onTap: onTap}
	if res != nil {
		ib.img = canvas.NewImageFromResource(res)
		ib.img.FillMode = canvas.ImageFillContain
	} else {
		ib.img = canvas.NewImageFromResource(nil)
	}
	ib.ExtendBaseWidget(ib)
	return ib
}

func (b *ImageButton) CreateRenderer() fyne.WidgetRenderer {
	return &imageButtonRenderer{img: b.img, obj: b}
}

func (b *ImageButton) SetResource(res fyne.Resource) {
	b.res = res
	if b.img == nil {
		b.img = canvas.NewImageFromResource(res)
		b.img.FillMode = canvas.ImageFillOriginal
	} else {
		b.img.Resource = res
	}
	b.Refresh()
}

func (b *ImageButton) SetSize(s fyne.Size) {
	b.size = s
	b.Refresh()
}

func (b *ImageButton) Tapped(_ *fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *ImageButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *ImageButton) MouseIn(*desktop.MouseEvent) {
	b.mu.Lock()
	b.hover = true
	b.mu.Unlock()
	fmt.Println("ImageButton: MouseIn")
	b.Refresh()
}

func (b *ImageButton) MouseMoved(*desktop.MouseEvent) {}

func (b *ImageButton) MouseOut() {
	b.mu.Lock()
	b.hover = false
	b.mu.Unlock()
	fmt.Println("ImageButton: MouseOut")
	b.Refresh()
}

type imageButtonRenderer struct {
	img *canvas.Image
	obj *ImageButton
}

func (r *imageButtonRenderer) Layout(size fyne.Size) {
	r.obj.mu.RLock()
	hover := r.obj.hover
	pref := r.obj.size
	r.obj.mu.RUnlock()

	scale := 0.85
	if hover {
		scale = 1.0
	}

	var w, h float32
	if pref.Width > 0 || pref.Height > 0 {
		w = float32(pref.Width) * float32(scale)
		h = float32(pref.Height) * float32(scale)
	} else {
		w = float32(size.Width) * float32(scale)
		h = float32(size.Height) * float32(scale)
	}
	r.img.Resize(fyne.NewSize(w, h))
	r.img.Move(fyne.NewPos((size.Width-r.img.Size().Width)/2, (size.Height-r.img.Size().Height)/2))
}

func (r *imageButtonRenderer) MinSize() fyne.Size {
	if r.obj.size.Width > 0 || r.obj.size.Height > 0 {
		return r.obj.size
	}
	return r.img.MinSize()
}

func (r *imageButtonRenderer) Refresh() {
	r.obj.mu.RLock()
	hover := r.obj.hover
	pref := r.obj.size
	r.obj.mu.RUnlock()

	scale := 0.95
	if hover {
		scale = 1.05
	}

	var w, h float32
	if pref.Width > 0 || pref.Height > 0 {
		w = float32(pref.Width) * float32(scale)
		h = float32(pref.Height) * float32(scale)
	} else {
		sz := r.obj.Size()
		w = float32(sz.Width) * float32(scale)
		h = float32(sz.Height) * float32(scale)
	}

	r.img.Resize(fyne.NewSize(w, h))
	r.img.Move(fyne.NewPos((r.obj.Size().Width-r.img.Size().Width)/2, (r.obj.Size().Height-r.img.Size().Height)/2))
	r.img.Refresh()
}

func (r *imageButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.img}
}

func (r *imageButtonRenderer) Destroy() {}

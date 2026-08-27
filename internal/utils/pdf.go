package utils

import (
	"bytes"
	"fmt"
	"strings"
)

// RGB is a fill/stroke color with components in the 0..1 range.
type RGB [3]float64

// Standard PDF font resource names set up by RenderPDF.
const (
	FontRegular = "F1" // Helvetica
	FontBold    = "F2" // Helvetica-Bold
	FontItalic  = "F3" // Helvetica-Oblique
)

// GradientSpec describes a two-stop linear (axial) gradient, in the same
// page-coordinate space used for drawing, from (X0,Y0) [C0] to (X1,Y1) [C1].
type GradientSpec struct {
	X0, Y0, X1, Y1 float64
	C0, C1         RGB
}

// PDFCanvas accumulates drawing operators (text, lines, filled shapes, a
// linear gradient) for a single-page PDF rendered with the standard
// Helvetica font family, which every PDF reader supports without the font
// needing to be embedded. It exists so a short, styled document (like a
// ticket) can be produced without pulling in a third-party PDF library.
type PDFCanvas struct {
	content bytes.Buffer
}

// NewPDFCanvas returns an empty canvas ready for drawing.
func NewPDFCanvas() *PDFCanvas {
	return &PDFCanvas{}
}

// SetFillColor sets the color used by FillRect/FillCircle/Text.
func (c *PDFCanvas) SetFillColor(color RGB) {
	fmt.Fprintf(&c.content, "%.3f %.3f %.3f rg\n", color[0], color[1], color[2])
}

// SetStrokeColor sets the color used by Line.
func (c *PDFCanvas) SetStrokeColor(color RGB) {
	fmt.Fprintf(&c.content, "%.3f %.3f %.3f RG\n", color[0], color[1], color[2])
}

// SetLineWidth sets the stroke width, in points, used by Line.
func (c *PDFCanvas) SetLineWidth(w float64) {
	fmt.Fprintf(&c.content, "%.2f w\n", w)
}

// SetDash sets a dash pattern (on/off, in points) for subsequent strokes;
// pass 0 to draw solid lines again.
func (c *PDFCanvas) SetDash(on, off float64) {
	if on <= 0 {
		c.content.WriteString("[] 0 d\n")
		return
	}
	fmt.Fprintf(&c.content, "[%.1f %.1f] 0 d\n", on, off)
}

// FillRect fills a rectangle with its origin at the bottom-left corner.
func (c *PDFCanvas) FillRect(x, y, w, h float64) {
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f re f\n", x, y, w, h)
}

// GradientRect fills a rectangle using the page's linear gradient pattern
// (set up via the GradientSpec passed to RenderPDF).
func (c *PDFCanvas) GradientRect(x, y, w, h float64) {
	c.content.WriteString("/Pattern cs /P1 scn\n")
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f re f\n", x, y, w, h)
}

// Line strokes a straight line from (x1,y1) to (x2,y2).
func (c *PDFCanvas) Line(x1, y1, x2, y2 float64) {
	fmt.Fprintf(&c.content, "%.2f %.2f m %.2f %.2f l S\n", x1, y1, x2, y2)
}

// FillCircle fills a circle of radius r centered at (cx,cy), approximated
// with four cubic Bezier curves. Used for the ticket's perforation notches.
func (c *PDFCanvas) FillCircle(cx, cy, r float64) {
	k := 0.5523 * r
	fmt.Fprintf(&c.content, "%.2f %.2f m\n", cx+r, cy)
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f %.2f %.2f c\n", cx+r, cy+k, cx+k, cy+r, cx, cy+r)
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f %.2f %.2f c\n", cx-k, cy+r, cx-r, cy+k, cx-r, cy)
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f %.2f %.2f c\n", cx-r, cy-k, cx-k, cy-r, cx, cy-r)
	fmt.Fprintf(&c.content, "%.2f %.2f %.2f %.2f %.2f %.2f c\n", cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	c.content.WriteString("f\n")
}

// Text draws left-aligned text with its baseline at (x, y) in the given
// font (FontRegular/FontBold/FontItalic), size and color.
func (c *PDFCanvas) Text(font string, size float64, color RGB, x, y float64, text string) {
	c.text(font, size, 0, color, x, y, text)
}

// TextSpaced is Text with extra space added between characters, for a
// letter-spaced label/badge look.
func (c *PDFCanvas) TextSpaced(font string, size, spacing float64, color RGB, x, y float64, text string) {
	c.text(font, size, spacing, color, x, y, text)
}

func (c *PDFCanvas) text(font string, size, spacing float64, color RGB, x, y float64, text string) {
	fmt.Fprintf(&c.content,
		"%.3f %.3f %.3f rg\nBT\n/%s %.2f Tf\n%.2f Tc\n1 0 0 1 %.2f %.2f Tm\n(%s) Tj\nET\n",
		color[0], color[1], color[2], font, size, spacing, x, y, escapePDFText(text))
}

// escapePDFText prepares a string for use inside a PDF literal string ( ... ),
// escaping the characters that are meaningful to the PDF syntax and dropping
// anything outside the standard font's single-byte encoding.
func escapePDFText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n' || r == '\r':
			b.WriteByte(' ')
		case r > 255:
			b.WriteByte('?')
		default:
			b.WriteByte(byte(r))
		}
	}
	return b.String()
}

// RenderPDF assembles a complete single-page PDF file (header, objects,
// xref table, trailer) of the given size around the canvas's content
// stream, with a linear-gradient pattern ("/P1") available to it as
// described by grad.
func RenderPDF(canvas *PDFCanvas, width, height float64, grad GradientSpec) []byte {
	contentStream := canvas.content.Bytes()

	var buf bytes.Buffer
	var offsets [11]int // index 1..10 used, one per object

	buf.WriteString("%PDF-1.4\n")

	writeObj := func(n int, body string) {
		offsets[n] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] "+
			"/Resources << /Font << /F1 4 0 R /F2 5 0 R /F3 6 0 R >> /Pattern << /P1 9 0 R >> >> "+
			"/Contents 10 0 R >>",
		width, height))
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>")
	writeObj(6, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique >>")
	writeObj(7, fmt.Sprintf(
		"<< /FunctionType 2 /Domain [0 1] /C0 [%.3f %.3f %.3f] /C1 [%.3f %.3f %.3f] /N 1 >>",
		grad.C0[0], grad.C0[1], grad.C0[2], grad.C1[0], grad.C1[1], grad.C1[2]))
	writeObj(8, fmt.Sprintf(
		"<< /ShadingType 2 /ColorSpace /DeviceRGB /Coords [%.2f %.2f %.2f %.2f] /Function 7 0 R /Extend [true true] >>",
		grad.X0, grad.Y0, grad.X1, grad.Y1))
	writeObj(9, "<< /Type /Pattern /PatternType 2 /Shading 8 0 R >>")

	offsets[10] = buf.Len()
	fmt.Fprintf(&buf, "10 0 obj\n<< /Length %d >>\nstream\n", len(contentStream))
	buf.Write(contentStream)
	buf.WriteString("\nendstream\nendobj\n")

	xrefStart := buf.Len()
	buf.WriteString("xref\n0 11\n0000000000 65535 f \n")
	for n := 1; n <= 10; n++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[n])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size 11 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", xrefStart)

	return buf.Bytes()
}

package route

import (
	"fmt"
	"math"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func (o *Route) Chart() {
	p := plot.New()

	p.Title.Text = "exp(x)"
	p.X.Label.Text = "x"
	p.Y.Label.Text = "exp(x)"

	const numPoints = 100
	pts := make(plotter.XYs, numPoints)
	for i := 0; i < numPoints; i++ {
		x := float64(i) / 1000 // x ranges from 0 to 0.1, 100/1000
		pts[i].X = x
		// pts[i].Y = math.Exp(math.Pow(x, 3.0/4))
		pts[i].Y = math.Exp(pow34(x))
		// pts[i].Y = math.Exp(x)
	}
	line, err := plotter.NewLine(pts)
	if err != nil {
		fmt.Println("Error creating line plotter:", err)
		return
	}
	p.Add(line)

	if err := p.Save(6*vg.Inch, 4*vg.Inch, "exp.png"); err != nil {
		fmt.Println("Error saving plot:", err)
		os.Exit(1)
	}
	fmt.Println("Plot saved to exp.png")
}

func pow34(x float64) float64 {
	const (
		a0 = +0.006923987
		a1 = +2.563897835
		a2 = -14.621961493
		a3 = +61.316361623
	)
	if x < 0.005 {
		return math.Sqrt(x * math.Sqrt(x))
	}
	return a0 + x*(a1+x*(a2+x*a3))
}

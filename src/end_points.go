package main

import (
	"fmt"
	"image/color"
	"net/http"
	"time"

	databox "github.com/me-box/lib-go-databox"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

func getStatusEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("active\n"))
}

func getLoadPlot(w http.ResponseWriter, req *http.Request) {

	loadAverage1Stats.RLock()
	loadAverage5Stats.RLock()
	loadAverage15Stats.RLock()

	xys1 := make(plotter.XYs, len(loadAverage1Stats.data))
	xys5 := make(plotter.XYs, len(loadAverage5Stats.data))
	xys15 := make(plotter.XYs, len(loadAverage15Stats.data))
	for i, d := range loadAverage1Stats.data {
		xys1[i].X = loadAverage1Stats.timestamp[i]
		xys1[i].Y = d
	}
	for i, d := range loadAverage5Stats.data {
		xys5[i].X = loadAverage5Stats.timestamp[i]
		xys5[i].Y = d
	}
	for i, d := range loadAverage15Stats.data {
		xys15[i].X = loadAverage15Stats.timestamp[i]
		xys15[i].Y = d
	}

	loadAverage1Stats.RUnlock()
	loadAverage5Stats.RUnlock()
	loadAverage15Stats.RUnlock()

	g := plotter.NewGrid()
	g.Horizontal.Color = color.RGBA{R: 220, B: 220, G: 220, A: 255}
	g.Vertical.Width = 0

	p, err := plot.New()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}

	l1, _ := plotter.NewLine(xys1)
	l1.Color = color.RGBA{R: 255, A: 255}
	l5, _ := plotter.NewLine(xys5)
	l5.Color = color.RGBA{B: 255, A: 255}
	l15, _ := plotter.NewLine(xys15)
	l15.Color = color.RGBA{G: 255, A: 255}

	p.Add(g, l1, l5, l15)
	p.Title.Text = "Load Average"
	p.Y.Label.Text = "Load"
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04:05"}
	p.Legend.Add("1 min", l1)
	p.Legend.Add("5 min", l5)
	p.Legend.Add("15 min", l15)

	wt, err := p.WriterTo(800, 400, "png")
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
	_, err = wt.WriteTo(w)
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
}

func getStats(w http.ResponseWriter, req *http.Request) {

	fmt.Println("[getStats] called ....")

	_csc := databox.NewDefaultCoreStoreClient(DATABOX_ZMQ_ENDPOINT)

	tMinus5mins := (time.Now().Unix() - (60 * 5)) * 1000

	memMin, err := _csc.TSJSON.Since(dataSourceFreememStructured.DataSourceID, tMinus5mins, databox.TimeSeriesQueryOptions{
		AggregationFunction: databox.Min,
	})
	if err != nil {
		fmt.Println("Error getting Max dataSourceFreememStructured", err.Error())
	}

	memMax, err := _csc.TSJSON.Since(dataSourceFreememStructured.DataSourceID, tMinus5mins, databox.TimeSeriesQueryOptions{
		AggregationFunction: databox.Max,
	})
	if err != nil {
		fmt.Println("Error getting Min dataSourceFreememStructured", err.Error())
	}

	loadMin, err := _csc.TSJSON.Since(dataSourceLoadavg1Structured.DataSourceID, tMinus5mins, databox.TimeSeriesQueryOptions{
		AggregationFunction: databox.Min,
	})
	if err != nil {
		fmt.Println("Error getting Max dataSourceFreememStructured", err.Error())
	}

	loadMax, err := _csc.TSJSON.Since(dataSourceLoadavg1Structured.DataSourceID, tMinus5mins, databox.TimeSeriesQueryOptions{
		AggregationFunction: databox.Max,
	})
	if err != nil {
		fmt.Println("Error getting Min dataSourceFreememStructured", err.Error())
	}

	/*loadSD, err := _tsc.Since(dataSourceLoadavg1Structured.DataSourceID, tMinus5mins, databox.JSONTimeSeriesQueryOptions{
		AggregationFunction: databox.StandardDeviation,
	})
	if err != nil {
		fmt.Println("Error getting Min dataSourceFreememStructured", err.Error())
	}*/

	fmt.Println(string(memMin),
		string(memMax),
		string(loadMin),
		string(loadMax))

	fmt.Fprintf(w,
		`{
			"mem":{"min":%s,"max":%s},
			 "load":{"min":%s,"max":%s}
		}`,
		string(memMin),
		string(memMax),
		string(loadMin),
		string(loadMax),
		//string(loadSD),
	)

}

func getMemPlot(w http.ResponseWriter, req *http.Request) {

	memStats.RLock()

	xys := make(plotter.XYs, len(memStats.data))
	for i, d := range memStats.data {
		xys[i].X = memStats.timestamp[i]
		xys[i].Y = d / float64(1048576)
	}

	memStats.RUnlock()

	g := plotter.NewGrid()
	g.Horizontal.Color = color.RGBA{R: 220, B: 220, G: 220, A: 255}
	g.Vertical.Width = 0

	p, err := plot.New()
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}

	lmem, _ := plotter.NewLine(xys)
	lmem.Color = color.RGBA{R: 255, A: 255}

	p.Add(g, lmem)
	p.Title.Text = "Free Mem"
	p.Y.Label.Text = "Mem (MB)"
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "15:04:05"}
	p.Legend.Add("Mem", lmem)

	wt, err := p.WriterTo(800, 400, "png")
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
	_, err = wt.WriteTo(w)
	if err != nil {
		fmt.Println(err)
		w.Write([]byte("Error\n"))
		return
	}
}

func getUI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", `
		<h1>Databox stats</h1>
		<center>
			<div style="width:93%">
				<img src="/app-os-monitor/ui/mem.png?rand=0" style="float:left; padding:1%; width:45%; min-width:500px">
				<img src="/app-os-monitor/ui/load.png?rand=0" style="float:left; padding:1%; width:45%; min-width:500px">
			</div>
		</center>
		<center>
			<div style="width:93%;float:left">
				<h2>Stats for last 5 minuets</h2>
				<span>Min Mem:</span><span id="minmem"></span><br/>
				<span>Max Mem:</span><span id="maxmem"></span><br/>
				<span>Min Load:</span><span id="minload"></span><br/>
				<span>Max Load:</span><span id="maxload"></span><br/>
			</div>
		</center>
		<script>
		setInterval(function() {
			let imgs = document.getElementsByTagName("IMG");

			//update first image (split to stop requests happening in parallel)
			var eqPos = imgs[0].src.lastIndexOf("=");
			var src = imgs[0].src.substr(0, eqPos+1);
			imgs[0].src = src + Math.random();

			let xhttp = new XMLHttpRequest();
			xhttp.open("GET","/app-os-monitor/ui/stats",true);
			xhttp.onreadystatechange = function() {
				if (this.readyState == 4 && this.status == 200) {
					let data = JSON.parse(this.responseText);
					document.getElementById("minmem").innerHTML = (data.mem.min.result / 1048576) + " MB";
					document.getElementById("maxmem").innerHTML = (data.mem.max.result / 1048576) + " MB";
					document.getElementById("minload").innerHTML = data.load.min.result;
					document.getElementById("maxload").innerHTML = data.load.max.result;
				}
			  };
			xhttp.send();

			//update second image (split to stop requests happening in parallel)
			var eqPos = imgs[1].src.lastIndexOf("=");
			var src = imgs[1].src.substr(0, eqPos+1);
			imgs[1].src = src + Math.random();

		}, 5000);
		</script>
	`)
}

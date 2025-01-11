
bikeride ride.json -prof -gpx cazalla.gpx
go tool pprof --text --cum bikeride.exe cpu.pprof > pprof.txt
go tool pprof --tree  bikeride.exe cpu.pprof > pprof.tree

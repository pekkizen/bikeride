
 Copyright 2022 Pekkizen. All rights reserved."
 Use of Bikeride is governed by GNU General Public License v3.0."

## Bikeride simulates a bicycle ride from a GPX format route/track file

The main results are total ride time and rider's energy comsumption.

### 1. Introduction

Bikeride uses standard Newtonian physics for calculations and doesn't make any compromizes or shortcuts for simpler or faster program code. Bikeride is still quite fast, calculating ~2000 routepoints in 80 km route in ~1 ms. Bikeride is now available as a 64-bit Windows program, but Linux a version might be coming soon.

Bikeride is wind aware: air drag forces are calculated for a given wind speed and direction. This can be used for simulating drive time and energy comsumption variation in different wind conditions. Good road elevation data is also essential for accurate results. The accuracy of elevation data depends of the source of the gpx track data. Elevations recorded from gps devices' gps-elevations are the worst. Gps device with baromethric altimeter may give quite exact elevations. Elevation data from bicycle routing services varies. Quite good elevation data offers e.g Brouter <https://brouter.de/brouter-web>. Brouter properly interpolates elevations from nearby elevations. Elevation data from any source is generally not smooth enough to represent real roads and must be more or less smoothed reducing elevation gain. Bikeride has a set of smoothing and filtering algorithms for this.

### 2. Use of Bikeride

Bikeride is implemented as a command line program, which is used as

_**bikeride ride-param-file | -cfg config-param-file | -gpx route-gpx-file**_

The ride-param-file includes parameters for the ride and the config-param-file has parameters for calculation control, methods etc. The files may contain same parameters: config-param-file parameters are taken in first and the ride-param-file parameters second. Parameters values in rideparam-file are prioritized and parameters in config-param-file are overun by them. Both parameter files are json format files, which can be edited by any text editor. Preferably use json format aware editor to ensure correct json syntax. No json syntax errors are accepted. If config-param-file if not given by -cfg, file config.json is read from the same location from which bikeride.exe was launched.

### 3.Introduction to results

Bikeride produces routename_results.txt file as a summary of the simulation. A part of this file is presented and commented below. Bikeride also produces a detailed per road segment route file of what really happened. This file is a human readable standard comma separated file, which can be plotted or explored by standard statistical, spreadsheet etc. tools. The route file has columns for e.g. entry and exit speeds of segment, target power,  power used by rider and braking, original GPX-file elevation and bikeride smoothed elevation and info of how the segment was ridden: ac/decelerating, braking, riding constant speed of or any combination of these.

#### 3.1 Rider speed and energy relalated results

| Rider energy usage     |  Wh  |  %     |
|------------------------|:-----:|:-----:|
| Total                  | 468  |    100  
| Gravity up             | 319  |  65.7  |
| Air resistance         |  62  |  12.8  |
| Rolling resistance     |  75  |  15.4  |
| Drivetrain loss        |  29  |  6.0   |
| Energy net sum         | **-0.0** |        |
| Acceleration           |  25  | (incl.above)
  Average power (W)      |  109 |
| Energy/distance (Wh/km)| 7.01 |

<https://fineli.fi/fineli/en> "Lard is a semi-solid white fat product obtained by rendering
the fatty tissue of a pig."

| Rider's food consumption          |         |  
|-----------------------------------|:--------:|
|  Food (kcal)                      | 1741    |
|  Bananas (pcs)                    |   16    |
|  Lard (g)                         |  203    |  

| Bicycle Speed      |  (km/h)   |
|---------------------|---------:|
| Mean                |  14.89   |
| Max                 |  39.98   |
| Min                 |  3.04    |
| Downhill < -4% mean |  28.94   |
| Down vertical (m/h) |  2073    |

| Total energy balance |       |   (Wh)   |
|----------------------|------:|---------:|
|  Rider               |       |  486     |
|  Drivetrain loss     |       |  -29     |
| Kinetic resistance   |       |  -104.7  |
|  Kinetic push        |       |  103.9   |
|  Gravity up          |       |  -324    |
|  Gravity down        |       |  328     |
|  Air resistance      |       |  -224    |
|<up><li>pedaling      | -82   |          |
|<up><li>freewheeling  | -105  |          |
|<up><li>braking       | -37   |          |
|  Rolling resistance  |       | -125     |
|  Braking             |       | -111     |
|  Energy net sum      |       | **-0.1** |

#### 3.2 Route relalated results

### 4 Summary of methods used

### 5 Introduction to Bikeride parameters

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"image/color"
	"io/ioutil"
	"path/filepath"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
)

// Functions

// ParseDataFile takes in a location to a test log file,
// reads its content, parses it and returns the relevant
// and needed parts of it.
func ParseDataFile(filePath string) (string, string, string, plotter.XYs, error) {

	// Consume data from specified file.
	dataRaw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", "", nil, err
	}

	// Parse data into usable format.
	data := strings.Split(string(dataRaw), "-----\n")

	// Prepare parts for further data extraction.
	data[0] = strings.TrimSpace(data[0])
	dataHeader := strings.Split(data[0], "\n")

	data[1] = strings.TrimSpace(data[1])
	dataPointsRaw := strings.Split(data[1], "\n")

	// Save meta information for direct access.
	dataSubject := strings.TrimPrefix(dataHeader[0], "Subject: ")
	dataPlatform := strings.TrimPrefix(dataHeader[1], "Platform: ")
	dataDateRaw := strings.TrimPrefix(dataHeader[2], "Date: ")

	// Reserve space for final data point slice of set one.
	dataPoints := make(plotter.XYs, len(dataPointsRaw))

	for i := range dataPointsRaw {

		// Split each point at comma.
		point := strings.Split(dataPointsRaw[i], ", ")

		// Convert string ID to float.
		id, err := strconv.ParseFloat(point[0], 64)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("failed to convert string ID to float.")
		}

		// Convert string value to float.
		value, err := strconv.ParseFloat(point[1], 64)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("failed to convert string value to float.")
		}

		// Scale value from nanoseconds to milliseconds.
		value = value / float64(1000000)

		// Append new data point to result slice.
		dataPoints[i].X = id
		dataPoints[i].Y = value
	}

	return dataSubject, dataPlatform, dataDateRaw, dataPoints, nil
}

// ParseDataFolder takes in a path to a folder containing
// measurements from a concurrent command test. It parses
// all files, combining and averaging each individual run
// over all available concurrently executed commands.
func ParseDataFolder(folderPath string) (string, string, string, plotter.XYs, error) {

	var dataSubject string
	var dataPlatform string
	var dataDateRaw string
	var dataPoints plotter.XYs

	// Find all files in supplied folder.
	files, err := filepath.Glob(filepath.Join(folderPath, "*"))
	if err != nil {
		return "", "", "", nil, err
	}

	// Save number of files (= concurrent accesses)
	// for later correction of data points.
	numFiles := float64(len(files))

	for i, file := range files {

		// Parse contents of current file.
		curDataSubject, curDataPlatform, curDataDateRaw, curDataPoints, err := ParseDataFile(file)
		if err != nil {
			return "", "", "", nil, err
		}

		if i == 0 {

			// Initially, set comparision values.
			dataSubject = curDataSubject
			dataPlatform = curDataPlatform
			dataDateRaw = curDataDateRaw
			dataPoints = curDataPoints

		} else {

			// Check for files being from the same test run.
			if (dataSubject != curDataSubject) || (dataPlatform != curDataPlatform) || (dataDateRaw != curDataDateRaw) {
				return "", "", "", nil, fmt.Errorf("files from same folder were not from same test")
			}

			for u := range dataPoints {

				// Add measured rtt from current data points set
				// to already accumulated set.
				dataPoints[u].Y += curDataPoints[u].Y
			}
		}
	}

	for u := range dataPoints {

		// Normalize each accumulated run by averaging
		// it over all performed runs.
		dataPoints[u].Y = dataPoints[u].Y / numFiles
	}

	return dataSubject, dataPlatform, dataDateRaw, dataPoints, nil
}

// PreparePlot initializes a new plot with acceptable
// styling defaults and returns it.
func PreparePlot(title string, xMax float64, xLabel string, yLabel string) (*plot.Plot, error) {

	// Use Helvetica in big size for title.
	bigFont, err := vg.MakeFont("Helvetica", 16)
	if err != nil {
		return nil, err
	}

	// Use Courier in small size for all labels.
	smallFont, err := vg.MakeFont("Courier", 11)
	if err != nil {
		return nil, err
	}

	// Create an empty plot.
	p, err := plot.New()
	if err != nil {
		return nil, err
	}

	// Set title of plot and its style.
	p.Title.Text = title
	p.Title.Padding = 1 * vg.Centimeter
	p.Title.Font = bigFont

	// Style x-axis a bit.
	p.X.Min = 1.0
	p.X.Max = xMax
	p.X.Label.Text = xLabel
	p.X.Label.Font = smallFont
	p.X.Padding = 0.2 * vg.Centimeter
	p.X.Tick.Label.Font = smallFont

	// Style y-axis a bit.
	p.Y.Min = 0.0
	p.Y.Label.Text = yLabel
	p.Y.Label.Font = smallFont
	p.Y.Padding = 0.1 * vg.Centimeter
	p.Y.Tick.Label.Font = smallFont

	// Style legend a bit.
	p.Legend.Font = smallFont

	return p, nil
}

// Usage prints out how to use this script with the two possible
// options: plot two files against each other or two folders.
// It exits the program.
func Usage() {

	// Print usage example and exit.
	fmt.Printf("Please specify either two test log files or two test log folders to plot against each other.\nFor example:\n")
	fmt.Printf("\t$ ./plot-results -fileOne results/pluto-append.log -fileTwo results/dovecot-append.log\n")
	fmt.Printf("\t$ ./plot-results -folderOne results/pluto-concurrent-store -folderTwo results/dovecot-concurrent-store\n")

	os.Exit(1)
}

func main() {

	var dataOneSubject, dataTwoSubject string
	var dataOnePlatform, dataTwoPlatform string
	var dataOneDateRaw, dataTwoDateRaw string
	var dataOnePoints, dataTwoPoints plotter.XYs
	var err error

	// Require two files or two folders to be plotted.
	fileOnePath := flag.String("fileOne", "", "Supply first log file of IMAP command test.")
	fileTwoPath := flag.String("fileTwo", "", "Supply second log file of IMAP command test.")
	folderOnePath := flag.String("folderOne", "", "Supply first log folder of concurrent IMAP command test.")
	folderTwoPath := flag.String("folderTwo", "", "Supply second log folder of concurrent IMAP command test.")
	flag.Parse()

	// Check that two files or two folders were supplied.
	if (*fileOnePath == "") && (*fileTwoPath == "") && (*folderOnePath == "") && (*folderTwoPath == "") {
		Usage()
	}

	if (*fileOnePath != "") && (*fileTwoPath != "") {

		// Parse data from first log file.
		dataOneSubject, dataOnePlatform, dataOneDateRaw, dataOnePoints, err = ParseDataFile(*fileOnePath)
		if err != nil {
			fmt.Printf("Failed to parse data from first file: %s\n", err.Error())
			os.Exit(1)
		}

		// Parse data from second log file.
		dataTwoSubject, dataTwoPlatform, dataTwoDateRaw, dataTwoPoints, err = ParseDataFile(*fileTwoPath)
		if err != nil {
			fmt.Printf("Failed to parse data from second file: %s\n", err.Error())
			os.Exit(1)
		}

	} else if (*folderOnePath != "") && (*folderTwoPath != "") {

		// Parse data from first log folder.
		dataOneSubject, dataOnePlatform, dataOneDateRaw, dataOnePoints, err = ParseDataFolder(*folderOnePath)
		if err != nil {
			fmt.Printf("Failed to parse data from first folder: %s\n", err.Error())
			os.Exit(1)
		}
		dataOneSubject = strings.Replace(dataOneSubject, " ", "-", -1)

		// Parse data from second log folder.
		dataTwoSubject, dataTwoPlatform, dataTwoDateRaw, dataTwoPoints, err = ParseDataFolder(*folderTwoPath)
		if err != nil {
			fmt.Printf("Failed to parse data from second folder: %s\n", err.Error())
			os.Exit(1)
		}
		dataTwoSubject = strings.Replace(dataTwoSubject, " ", "-", -1)

	} else {

		// Wrong constellation of arguments.
		// Print examples and exit.
		Usage()
	}

	// Check if tests ran the same command.
	if dataOneSubject != dataTwoSubject {
		fmt.Printf("Tests ran different commands.\n")
		os.Exit(1)
	}

	// If they were the same, set joint subject.
	dataSubject := dataOneSubject

	// Set x-axis maximum to bigger of two set values.
	var xMax float64
	if len(dataOnePoints) > len(dataTwoPoints) {
		xMax = float64(len(dataOnePoints))
	} else {
		xMax = float64(len(dataTwoPoints))
	}

	// Prepare time of first test file for printing.
	dataOneDate, err := time.Parse("2006-01-02-15-04-05", dataOneDateRaw)
	if err != nil {
		fmt.Printf("Failed to format date of first log file: %s\n", err.Error())
		os.Exit(1)
	}

	// Prepare time of second test file for printing.
	dataTwoDate, err := time.Parse("2006-01-02-15-04-05", dataTwoDateRaw)
	if err != nil {
		fmt.Printf("Failed to format date of second log file: %s\n", err.Error())
		os.Exit(1)
	}

	// Construct a title for output plot.
	title := fmt.Sprintf("Command %s: %s (%s) vs. %s (%s)", dataSubject, dataOnePlatform, dataOneDate.Format("2006-01-02 15:04:05"), dataTwoPlatform, dataTwoDate.Format("2006-01-02 15:04:05"))

	// Now create a new plot with custom styling.
	p, err := PreparePlot(title, xMax, "Message number (id)", "Completion time (ms)")
	if err != nil {
		fmt.Printf("Failed to initialize new plot: %s\n", err.Error())
		os.Exit(1)
	}

	// Add scatter plot based on data set one.
	scatterOne, err := plotter.NewScatter(dataOnePoints)
	if err != nil {
		fmt.Printf("Failed to add scatter plot of first log file to plot: %s\n", err.Error())
		os.Exit(1)
	}

	// Add scatter plot based on data set two.
	scatterTwo, err := plotter.NewScatter(dataTwoPoints)
	if err != nil {
		fmt.Printf("Failed to add scatter plot of second log file to plot: %s\n", err.Error())
		os.Exit(1)
	}

	// Let elements look like crosses and color differently.
	scatterOne.GlyphStyle.Shape = draw.CrossGlyph{}
	scatterOne.GlyphStyle.Color = color.RGBA{R: 0, G: 0, B: 184, A: 255}
	scatterTwo.GlyphStyle.Shape = draw.CrossGlyph{}
	scatterTwo.GlyphStyle.Color = color.RGBA{R: 239, G: 191, B: 0, A: 255}

	// Finally, add scatter plots to prepared canvas plot.
	p.Add(scatterOne, scatterTwo)
	p.Legend.Add(dataOnePlatform, scatterOne)
	p.Legend.Add(dataTwoPlatform, scatterTwo)

	// Save resulting plot to svg file.
	err = p.Save((9 * vg.Inch), (9 * vg.Inch), fmt.Sprintf("results/%s-on-%s-%s-vs-%s-%s.svg", dataSubject, dataOnePlatform, dataOneDateRaw, dataTwoPlatform, dataTwoDateRaw))
	if err != nil {
		fmt.Printf("Could not save finished plot to file: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("\nDone.\n")
}

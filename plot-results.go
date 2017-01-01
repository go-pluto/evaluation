package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"io/ioutil"

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
	dataSubject := strings.TrimLeft(dataHeader[0], "Subject: ")
	dataPlatform := strings.TrimLeft(dataHeader[1], "Platform: ")
	dataDateRaw := strings.TrimLeft(dataHeader[2], "Date: ")

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
	p.X.Min = 0.0
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

func main() {

	// Require two files to be plotted.
	filePath := flag.String("files", "", "Supply two space-separated test run log files for the same IMAP command.")
	flag.Parse()

	// Check that two files were supplied.
	if *filePath == "" {
		log.Fatalf("[evaluation.Plot] Please specify two space-separated test run log files to plot against each other.\n")
	}

	// Parse out the two files.
	files := strings.Split(*filePath, " ")

	// Check that actually two were supplied.
	if len(files) != 2 {
		log.Fatalf("[evaluation.Plot] Please supply exactly two corresponding test log files.\n")
	}

	// Parse data from first log file.
	dataOneSubject, dataOnePlatform, dataOneDateRaw, dataOnePoints, err := ParseDataFile(files[0])
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to parse data from first file: %s\n", err.Error())
	}

	// Parse data from second log file.
	dataTwoSubject, dataTwoPlatform, dataTwoDateRaw, dataTwoPoints, err := ParseDataFile(files[0])
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to parse data from first file: %s\n", err.Error())
	}

	// Check if tests ran the same command.
	if dataOneSubject != dataTwoSubject {
		log.Fatalf("[evaluation.Plot] Tests ran different commands.\n")
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
		log.Fatalf("[evaluation.Plot] Failed to format date of first log file: %s\n", err.Error())
	}

	// Prepare time of second test file for printing.
	dataTwoDate, err := time.Parse("2006-01-02-15-04-05", dataTwoDateRaw)
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to format date of second log file: %s\n", err.Error())
	}

	// Construct a title for output plot.
	title := fmt.Sprintf("Command %s: %s (%s) vs. %s (%s)", dataSubject, dataOnePlatform, dataOneDate.Format("2006-01-02 15:04:05"), dataTwoPlatform, dataTwoDate.Format("2006-01-02 15:04:05"))

	// Now create a new plot with custom styling.
	p, err := PreparePlot(title, xMax, "Message number (id)", "Completion time (ms)")
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to initialize new plot: %s\n", err.Error())
	}

	// Add scatter plot based on data set one.
	scatterOne, err := plotter.NewScatter(dataOnePoints)
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to add scatter plot of first log file to plot: %s\n", err.Error())
	}

	// Add scatter plot based on data set two.
	scatterTwo, err := plotter.NewScatter(dataTwoPoints)
	if err != nil {
		log.Fatalf("[evaluation.Plot] Failed to add scatter plot of second log file to plot: %s\n", err.Error())
	}

	// Let elements look like crosses.
	scatterOne.GlyphStyle.Shape = draw.CrossGlyph{}
	scatterTwo.GlyphStyle.Shape = draw.CrossGlyph{}

	// Finally, add scatter plots to prepared canvas plot.
	p.Add(scatterOne, scatterTwo)
	p.Legend.Add(dataOnePlatform, scatterOne)
	p.Legend.Add(dataTwoPlatform, scatterTwo)

	// Save resulting plot to svg file.
	err = p.Save((9 * vg.Inch), (9 * vg.Inch), fmt.Sprintf("%s-on-%s-%s-vs-%s-%s.svg", dataSubject, dataOnePlatform, dataOneDateRaw, dataTwoPlatform, dataTwoDateRaw))
	if err != nil {
		log.Fatalf("[evaluation.Plot] Could not save finished plot to file: %s\n", err.Error())
	}

	log.Printf("Done.\n")
}

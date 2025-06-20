package main

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gocv.io/x/gocv"
)

func main() {
	webcam, err := gocv.VideoCaptureDevice(0)
	if err != nil {
		fmt.Printf("Error opening webcam: %v\n", err)
		return
	}
	defer webcam.Close()

	window := gocv.NewWindow("Motion Detection")
	defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	gray := gocv.NewMat()
	defer gray.Close()

	previousGray := gocv.NewMat()
	defer previousGray.Close()

	diff := gocv.NewMat()
	defer diff.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	frameWidth := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	frameHeight := int(webcam.Get(gocv.VideoCaptureFrameHeight))
	fps := webcam.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 20.0 // fallback
	}

	fmt.Println("Start reading webcam")

	var writer *gocv.VideoWriter
	var writing bool
	var recordingStart time.Time

	cooldownUntil := time.Now()

	for {
		if ok := webcam.Read(&img); !ok || img.Empty() {
			continue
		}

		gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)
		gocv.GaussianBlur(gray, &gray, image.Pt(21, 21), 0, 0, gocv.BorderDefault)

		motionDetected := false

		if !previousGray.Empty() {
			gocv.AbsDiff(previousGray, gray, &diff)
			gocv.Threshold(diff, &diff, 25, 255, gocv.ThresholdBinary)
			gocv.Dilate(diff, &diff, kernel)

			contours := gocv.FindContours(diff, gocv.RetrievalExternal, gocv.ChainApproxSimple)
			for i := range contours.Size() {
				c := contours.At(i)
				if gocv.ContourArea(c) < 1500 {
					continue
				}
				motionDetected = true
				rect := gocv.BoundingRect(c)
				gocv.Rectangle(&img, rect, color.RGBA{0, 255, 0, 0}, 2)
				gocv.PutText(&img, "Motion Detected", image.Pt(10, 20), gocv.FontHersheyPlain, 1.5, color.RGBA{255, 0, 0, 0}, 2)
			}
		}

		// Start recording
		if motionDetected && !writing && time.Now().After(cooldownUntil) {
			filename := fmt.Sprintf("tmp/motion_%d.avi", time.Now().Unix())
			writer, err = gocv.VideoWriterFile(filename, "MJPG", fps, frameWidth, frameHeight, true)
			if err != nil {
				fmt.Println("Error creating video file:", err)
			} else {
				fmt.Println("Recording started:", filename)
				writing = true
				recordingStart = time.Now()
			}
		}

		// Continue writing video
		if writing {
			writer.Write(img)

			// Stop recording after 5 seconds
			if time.Since(recordingStart) > 5*time.Second {
				fmt.Println("Recording stopped")
				writer.Close()
				writer = nil
				writing = false
				cooldownUntil = time.Now().Add(10 * time.Second)
			}
		}

		gray.CopyTo(&previousGray)
		window.IMShow(img)
		if window.WaitKey(1) == 27 {
			break // ESC to quit
		}
	}

	if writer != nil {
		writer.Close()
	}
}

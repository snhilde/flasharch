package main

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// This is the mirror where we'll get the ISO. The full list of mirrors can be found on the main site here:
// https://www.archlinux.org/download/
var mirror = "https://mirrors.ocf.berkeley.edu/archlinux/iso/latest/"

var units = []string{"B", "K", "M", "G"}

func main() {
	if runtime.GOOS != "linux" {
		fmt.Println(os.Args[0], "has only been tested on Linux")
		os.Exit(1)
	}

	// Get the path to the USB drive, and perform some sanity checks.
	usb := getUSB()
	if usb == "" {
		os.Exit(1)
	}

	// Verify that the provided mirror URL is valid.
	u, err := url.Parse(mirror)
	if err != nil {
		fmt.Println("Error parsing mirror:", err)
		os.Exit(1)
	}
	url := u.String()
	fmt.Println("Looking for ISO in", url)

	// Get the filename of the ISO we want.
	filename := getFilename(url)
	if filename == "" {
		os.Exit(1)
	}

	// Use these paths to download and save the ISO.
	url += "/" + filename
	isoFile := os.TempDir() + "/" + filename

	// Download the ISO.
	fmt.Println("Downloading", filename, "...")
	if err := downloadFile(url, isoFile); err != nil {
		fmt.Println("Error downloading ISO:", err)
		os.Exit(1)
	}
	fmt.Printf("\n") // Flush last progress line.
	fmt.Println("Download complete")

	// Use these paths to download and save the ISO's signature.
	filename += ".sig"
	url += ".sig"
	sigFile := isoFile + ".sig"

	// Download the ISO's signature.
	fmt.Println("Downloading", filename, "...")
	if err := downloadFile(url, sigFile); err != nil {
		fmt.Println("Error downloading signature:", err)
		os.Exit(1)
	}
	fmt.Printf("\n") // Flush last progress line.
	fmt.Println("Download complete")

	// Verify the ISO with the signature.
	fmt.Println("Verifying ISO")
	cmd := exec.Command("gpg", "--keyserver-options", "auto-key-retrieve", "--verify", sigFile, isoFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Error verifying ISO:", err)
		os.Exit(1)
	} else {
		lines := strings.Split(string(output), "\n")
		for _, v := range lines {
			fmt.Println("\t", v)
		}
	}

	// Flash the ISO to the specified USB.
	fmt.Println("Flashing ISO to", usb)
	cmd = exec.Command("dd", "if="+isoFile, "of="+usb, "bs=1M", "status=progress")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Error flashing ISO:", err)
		os.Exit(1)
	} else {
		lines := strings.Split(string(output), "\n")
		for _, v := range lines {
			fmt.Println("\t", v)
		}
	}
	fmt.Println("Flash complete")

	// Clean up the temporary files we created.
	if err := os.Remove(isoFile); err != nil {
		fmt.Println("Error removing ISO file:", err)
		os.Exit(1)
	}
	if err := os.Remove(sigFile); err != nil {
		fmt.Println("Error removing signature file:", err)
		os.Exit(1)
	}
}

// getUSB checks the provided path to the USB drive and returns it back to the caller.
func getUSB() string {
	// Make sure the user provided a path to the USB drive.
	if len(os.Args) != 2 {
		if len(os.Args) < 2 {
			fmt.Println("Missing path to USB drive")
		} else {
			fmt.Println("Invalid arguments")
		}
		fmt.Println("Usage:")
		fmt.Println("\t", os.Args[0], "/full/path/to/usb")
		return ""
	}
	usb := os.Args[1]

	// Make sure we have an absolute path
	if !path.IsAbs(usb) {
		fmt.Println("Must use absolute path to USB drive")
		fmt.Println("Usage:")
		fmt.Println("\t", os.Args[0], "/full/path/to/usb")
		return ""
	}

	// Make sure the path is valid.
	info, err := os.Stat(usb)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	// Make sure we have write permissions to the USB. We can't really error out on the type assertion, so we'll only do
	// this additional sanity check if we can.
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		// Check if we match the file's user or group.
		isUser := os.Getuid() == int(stat.Uid)
		isGroup := os.Getgid() == int(stat.Gid)

		// Find out which of the file's user, group, and other write bits are set.
		perms := info.Mode().Perm() & os.ModePerm
		uWrite := perms&(1<<7) > 0
		gWrite := perms&(1<<4) > 0
		oWrite := perms&(1<<1) > 0

		if !(isUser && uWrite) && !(isGroup && gWrite) && !oWrite {
			fmt.Println("Cannot write to", usb)
			return ""
		}
	}

	return usb
}

// getFilename parses the mirror's directory and pulls out the name of the ISO file that we will download.
func getFilename(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error accessing mirror:", err)
		return ""
	}
	defer resp.Body.Close()

	// Parse the HTML data into a tree/doc.
	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Println("Error parsing mirror's directory:", err)
		return ""
	}

	// Move through the document until we find our ISO. We'll traverse the tree in this order of tags:
	tags := []string{"html", "body", "table", "tbody", "tr", "td", "a"}
	filename := parseBody(doc, tags)
	if filename == "" {
		fmt.Println("Mirror does not have the latest ISO")
		return ""
	}

	return filename
}

// parseBody parses the provided HTML and pulls out the name of the ISO that we want to download.
func parseBody(parent *html.Node, tags []string) string {
	if len(tags) == 0 {
		// We found a link tag. Let's see if it's pointing to an ISO.
		for _, a := range parent.Attr {
			if a.Key == "href" && strings.HasSuffix(a.Val, ".iso") {
				// We found it.
				return a.Val
			}
		}
		// Nothing yet.
		return ""
	}

	// Check each child node until we find an element with the desired tag.
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tags[0] {
			// We found the tag we want. Keep going down.
			if iso := parseBody(child, tags[1:]); iso != "" {
				return iso
			}
		}
	}

	// If we're here, then we didn't find the child that we were looking for. We'll move back up a level and keep trying.
	return ""
}

// downloadFile downloads the file at the url. In order to show a progress bar, we're going to wrap our HTTP response in
// a Tee Reader. This will allow us to monitor the number of bytes received in realtime. Thank you, Edd Turtle, for this
// recommendation.
func downloadFile(url, filename string) error {
	// Create a save point.
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Grab the file's data.
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Make sure we accessed everything correctly.
	if resp.StatusCode != 200 {
		return fmt.Errorf("%v", resp.Status)
	}

	// Set up our progress bar.
	p := progress{total: reduce(int(resp.ContentLength))}
	t := io.TeeReader(resp.Body, &p)

	// Save the file.
	_, err = io.Copy(file, t)

	return err
}

// Progress will be used to display a progress bar during the download operation.
type progress struct {
	total string // size of file to be downloaded, ready for printing
	have  int    // number of bytes we currently have
	count int    // running count of write operations, for determining if we should print or not
}

func (pr *progress) Write(p []byte) (int, error) {
	n := len(p)
	pr.have += n

	// We don't need to do expensive print operations that often.
	pr.count++
	if pr.count%50 > 0 {
		return n, nil
	}

	// Clear the line.
	fmt.Printf("\r%s", strings.Repeat(" ", 50))

	// Print the current transfer status.
	fmt.Printf("\rReceived %v of %v total", reduce(pr.have), pr.total)

	return n, nil
}

// reduce will convert the number of bytes into its human-readable value (less than 1024) with SI unit suffix appended.
func reduce(n int) string {
	index := int(math.Log2(float64(n))) / 10
	n >>= (10 * index)

	return strconv.Itoa(n) + units[index]
}

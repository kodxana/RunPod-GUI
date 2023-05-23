package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/cavaliergopher/grab/v3"
)

func isRunpodctlInstalled() bool {
	_, err := os.Stat("./runpodctl.exe")
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			log.Printf("Error checking file: %v", err)
			return false
		}
	}
	return true
}

func main() {
	app := app.New()

	win := app.NewWindow("runpodctl GUI")
	win.Resize(fyne.NewSize(1920, 1080))
	// Create the settings/navigation panel
	settingsBox := container.NewVBox()

	apiKeyEntry := widget.NewEntry()
	apiKeyEntry.SetPlaceHolder("Enter API Key")
	settingsBox.Add(apiKeyEntry)

	saveButton := widget.NewButton("Save", func() {
		// Save the API Key here
		apiKey := apiKeyEntry.Text

		// Prepare the command
		cmd := exec.Command("./runpodctl.exe", "config", "--apiKey="+apiKey)

		// Run the command and capture any error
		if err := cmd.Run(); err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Success", "API key configured successfully", win)
		}
	})
	settingsBox.Add(saveButton)

	versionButton := widget.NewButton("Version", func() {
		// Prepare the command
		cmd := exec.Command("./runpodctl.exe", "version")

		// Run the command and capture the output
		output, err := cmd.Output()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		// Display the output in a dialog box
		dialog.ShowInformation("Version", string(output), win)
	})
	settingsBox.Add(versionButton)

	// Create the file transfer panel
	transferBox := container.NewVBox()

	dataPath := widget.NewEntry()
	dataPath.SetPlaceHolder("Data path")

	pickFileButton := widget.NewButton("Select File", func() {
		fileDialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}

			// Set the dataPath text to the selected file path
			dataPath.SetText(file.URI().Path())

			go func() {
				// Prepare the command
				cmd := exec.Command("./runpodctl.exe", "send", file.URI().Path())

				// Run the command and capture the output
				outputBytes, err := cmd.Output()
				if err != nil {
					fyne.CurrentApp().SendNotification(fyne.NewNotification("Error", err.Error()))
					return
				}

				output := string(outputBytes)

				// Find the receive command in the output
				receiveCommandIndex := strings.Index(output, "runpodctl receive")
				if receiveCommandIndex == -1 {
					fyne.CurrentApp().SendNotification(fyne.NewNotification("Error", "Failed to find receive command in output"))
					return
				}

				// Extract the receive command
				receiveCommand := output[receiveCommandIndex:]

				// Display the output in a dialog box
				fyne.CurrentApp().SendNotification(fyne.NewNotification("Sent", receiveCommand))
			}()
		}, win)

		fileDialog.Show()

	})

	pickFolderButton := widget.NewButton("Select Folder", func() {
		var zipFilePath string
		folderDialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}

			folderPath := dir.String()[7:]
			zipFileName := filepath.Base(folderPath) + ".zip"

			// Get current directory
			currentDir, err := os.Getwd()
			if err != nil {
				dialog.ShowError(err, win)
				return
			}

			zipFilePath = filepath.Join(currentDir, zipFileName)

			go func() {
				// Create a new zip archive
				zipFile, err := os.Create(zipFileName)
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				defer zipFile.Close()

				zipWriter := zip.NewWriter(zipFile)
				defer zipWriter.Close()

				// Walk the directory and add each file to the zip
				filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil // skip directories
					}

					// Open the file
					file, err := os.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()

					// Create a new entry in the zip
					zipPath := strings.TrimPrefix(path, folderPath+"/") // remove base path
					writer, err := zipWriter.Create(zipPath)
					if err != nil {
						return err
					}

					// Write the file to the zip
					_, err = io.Copy(writer, file)
					return err
				})

				// Update dataPath label on the main thread
				dataPath.SetText(zipFilePath)
				dataPath.Refresh()
			}()
		}, win)

		folderDialog.SetOnClosed(func() {
			// Remove the zip file when the dialog is closed
			err := os.Remove(zipFilePath)
			if err != nil {
				fmt.Println(err)
			}
		})

		folderDialog.Show()
	})

	transferBox.Add(dataPath)
	transferBox.Add(pickFileButton)
	transferBox.Add(pickFolderButton)

	sendButton := widget.NewButton("Send", func() {
		if dataPath.Text == "" {
			dialog.ShowError(errors.New("please select a file or folder to send"), win)
			return
		}

		// Prepare the command
		cmd := exec.Command("./runpodctl.exe", "send", dataPath.Text)

		// Get stdout and stderr pipes
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		// Start the command
		err = cmd.Start()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		// Create receiveCommand binding
		receiveCommand := binding.NewString()
		// Prepare a label to display the output
		outputData := binding.NewString()
		outputEntry := widget.NewMultiLineEntry()
		outputEntry.SetPlaceHolder("Sending...")
		outputEntry.Wrapping = fyne.TextWrapBreak
		outputEntry.Disable()
		// Set text color
		outputEntry.TextStyle = fyne.TextStyle{Monospace: true, Bold: true, Italic: true}
		// Set up data binding
		outputEntry.Bind(outputData)

		// Read the command output in a goroutine
		go func() {
			reader := bufio.NewReader(stdoutPipe)
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Printf("Error reading stdout: %v", err)
					continue
				}

				// Log the output to the console
				log.Printf("stdout: %s", line)
			}

			// Wait for command to finish
			if err := cmd.Wait(); err != nil {
				log.Printf("Command finished with error: %v", err)
			}
		}()

		// Read the command error in a goroutine
		go func() {
			reader := bufio.NewReader(stderrPipe)
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Printf("Error reading stderr: %v", err)
					continue
				}

				// Log the output to the console
				log.Printf("stderr: %s", line)

				// Extract the generated code using regular expressions
				re := regexp.MustCompile(`runpodctl receive [A-Za-z0-9-]+`)
				match := re.Find(line)
				if match != nil {
					// The line contains the receive command
					_ = receiveCommand.Set(string(match))
					outputEntry.SetText("Run this command on another machine to get the file:\n" + string(match))
				}
			}

			// Notify the main goroutine about the completion and the receive command
			if cmd, err := receiveCommand.Get(); cmd != "" && err == nil {
				fyne.CurrentApp().SendNotification(fyne.NewNotification("Done", cmd))
			}
		}()

		// Create a new window to display the output
		outputWindow := app.NewWindow("Sending...")
		outputWindow.Resize(fyne.NewSize(800, 500))

		outputWindow.SetContent(container.NewVScroll(container.NewVBox(
			widget.NewLabel("Run this command on another machine to get the file:"),
			widget.NewLabelWithData(receiveCommand),
			widget.NewButton("Copy to Clipboard", func() {
				cmd, err := receiveCommand.Get()
				if err != nil {
					log.Printf("Error getting command: %v", err)
					return
				}
				outputWindow.Clipboard().SetContent(cmd)
			}),
		)))

		// Show the window
		outputWindow.Show()

		// Close the window and kill the process when the window is closed
		outputWindow.SetCloseIntercept(func() {
			outputWindow.Close()
			defer func() {
				err := cmd.Process.Kill()
				if err != nil {
					log.Printf("Error killing process: %v", err)
				}
			}()
		})
	})

	transferBox.Add(sendButton)

	receiveButton := widget.NewButton("Receive", func() {
		folderDialog := dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}

			folderPath := dir.String()[7:]

			// Create an input box for the receive code
			receiveCodeEntry := widget.NewEntry()

			// Prompt for receive code
			dialog.ShowCustomConfirm("Enter receive code", "Receive", "Cancel", receiveCodeEntry, func(ok bool) {
				if ok && receiveCodeEntry.Text != "" {
					go func() {
						// Get the current directory
						currentDir, err := os.Getwd()
						if err != nil {
							dialog.ShowError(err, win)
							return
						}

						// Change to the selected directory
						err = os.Chdir(folderPath)
						if err != nil {
							dialog.ShowError(err, win)
							return
						}

						// Create a command to run runpodctl.exe
						cmd := exec.Command(filepath.Join(currentDir, "runpodctl.exe"), "receive", receiveCodeEntry.Text)

						// Execute the command
						output, err := cmd.CombinedOutput()
						if err != nil {
							dialog.ShowError(fmt.Errorf("%s: %s", err, output), win)
							return
						}

						// Change back to the original directory
						err = os.Chdir(currentDir)
						if err != nil {
							dialog.ShowError(err, win)
							return
						}

						dialog.ShowInformation("Success", "Data received successfully.", win)
					}()
				}
			}, win)
		}, win)

		folderDialog.Show()
	})

	transferBox.Add(receiveButton)

	var installButton *widget.Button
	// Check if runpodctl.exe is already installed
	if !isRunpodctlInstalled() {
		installButton = widget.NewButton("Install runpodctl", func() {
			progressBar := widget.NewProgressBar()
			progressBar.Min = 0
			progressBar.Max = 100
			settingsBox.Add(progressBar)

			go func() {
				// Create client
				client := grab.NewClient()

				// Create request
				req, err := grab.NewRequest(".", "https://github.com/runpod/runpodctl/releases/download/v1.9.0/runpodctl-win-amd")
				if err != nil {
					log.Printf("Error creating request: %v", err)
					return
				}

				// Start download
				resp := client.Do(req)

				// Start UI updater
				t := time.NewTicker(500 * time.Millisecond)
				defer t.Stop()

				for {
					select {
					case <-t.C:
						// Update UI progress
						progressBar.SetValue(resp.Progress() * 100)
					case <-resp.Done:
						// Download completed, remove ticker
						t.Stop()

						// Update progress to 100%
						progressBar.SetValue(100)

						// Rename downloaded file to runpodctl.exe
						err := os.Rename(resp.Filename, "runpodctl.exe")
						if err != nil {
							log.Printf("Error renaming file: %v", err)
							return
						}

						dialog.ShowInformation("Success", "runpodctl installed successfully", win)

						// Remove progress bar
						settingsBox.Remove(progressBar)
						return
					}
				}
			}()
			// Disable the button to prevent multiple installations
			installButton.Disable()
		})
		settingsBox.Add(installButton)
	}

	// Create a horizontal split layout
	hSplit := container.NewHSplit(settingsBox, transferBox)

	win.SetContent(hSplit)
	win.ShowAndRun()
}

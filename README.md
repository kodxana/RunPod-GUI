# RunPod-GUI

RunPod-GUI is a user-friendly graphical user interface (GUI) for the `runpodctl` command-line interface (CLI) tool. 

## Features

- **Send/Receive Files and Folders**: Easily send and receive files and folders between different machines using a generated code.
- **Easy-to-use Interface**: Simplifies the process of sending and receiving files, making `runpodctl` accessible to users who are not familiar with CLI tools.
- **Automatic Installation**: The GUI will automatically install `runpodctl` if it is not already installed on your system.
- **Portable**: The application is fully portable and does not require an installation process.

## How to Use

1. **Send Data**: Click the "Select File" or "Select Folder" button to choose the file or folder you want to send. After selecting, click the "Send" button to generate a unique code. Share this code with the receiver.

2. **Receive Data**: Enter the unique code shared by the sender and select the folder where you want to save the received data. Click the "Receive" button to start receiving the data.

## Prerequisites

Before you begin, ensure that the `runpodctl.exe` file is placed in the same directory as the RunPod-GUI executable. If `runpodctl` is not installed on your system, the GUI will automatically handle the installation.

## Building from Source

If you're a developer and wish to build the project from the source code, follow these steps:

1. Clone the repository: `git clone https://github.com/kodxana/RunPod-GUI.git`
2. Navigate to the directory: `cd RunPod-GUI`
3. Install the required dependencies:
    - Go: Follow the instructions at https://golang.org/doc/install to install Go.
    - Fyne: Run `go get fyne.io/fyne/v2` to install Fyne.
4. Build the application: `go build -ldflags "-H windowsgui" main.go`

## Feedback

If you have any issues or suggestions, please feel free to open an issue in this repository.

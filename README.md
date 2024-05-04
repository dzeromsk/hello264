# hello264

This project is a very basic H264 encoder built in Go. The primary goal of this project was to gain an understanding of how H264 file format works. It utilizes the I_PCM block type, hence there's no compression of raw YUV420P data.

## Features

- Converts raw YUV420P data to H264 format.
- Uses I_PCM block format without compression.

## How it Works

The project takes raw YUV420P data and encodes it into the H264 format using the I_PCM block type. It segments the encoded data into NAL units.

## Getting Started

### Prerequisites

- Go toolchain
- YUV420P raw data input

### Installation

Install the package:

```bash
go install github.com/dzeromsk/hello264@latest
```

### Usage
To encode raw YUV420P data to H264:

```bash
hello264 < input.yuv > output.h264
```

Replace input.yuv with the path to your raw YUV420P data and output.h264 with the desired output file path.

## Notes
Inspired by 
* [World’s Smallest H.264 Encoder](https://www.cardinalpeak.com/blog/worlds-smallest-h-264-encoder) 
* [A minimal h264 “encoder”](https://jordicenzano.name/2014/08/31/the-source-code-of-a-minimal-h264-encoder-c/)

## License
This project is licensed under the Apache License - see the LICENSE file for details.

all: out.mp4

hello264: hello264.go
	go build -o hello264 hello264.go

in.yuv: testdata/in.mp4 Makefile
	ffmpeg -y -i testdata/in.mp4 -s 1280x720 -pix_fmt yuv420p in.yuv

out.264: hello264 in.yuv
	./hello264 < in.yuv > out.264

out.mp4: out.264
	ffmpeg -y -f h264 -i out.264 -vcodec copy out.mp4

clean:
	rm -f hello264 in.yuv out.264 out.mp4

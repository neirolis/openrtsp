# OpenRTSP with FFmpeg cameras module

A special solution where receiving a video stream from an IP camera is carried out using openRTSP and decoding the stream is performed by ffmpeg

## Install

```shell
go build -o openrtsp_ffmpeg
mkdir -p $RTMIPDIR/cameras/openrtsp
mv openrtsp_ffmpeg $RTMIPDIR/cameras/openrtsp/
cp manifest.yaml $RTMIPDIR/cameras/openrtsp/
```

Where `$RTMIPDIR` usually is a `/srv/rtmip`.

Or from the package [cameras-openrtsp.yaml](cameras-openrtsp.yaml).

## Manual Usage

```
usage: openrtsp_ffmpeg <addr> [--framerate=<n>] [--quality=<n>] [--archive=<s>]

positional:
  addr                    rtsp stream address

options:
      --framerate=<n>     target fps [default: 5]
      --quality=<n>       frames quality 1-31 [default: 3]
      --archive=<s>       archive type: mp4 only supported now
  -h, --help              display this help and exit
```

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	engine "plates_recognition_grpc"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

var (
	hostConfig = flag.String("host", "0.0.0.0", "server's hostname")
	portConfig = flag.String("port", "50051", "server's port")
	fileConfig = flag.String("file", "sample.jpg", "filename")

	xConfig = flag.String("x", "0", "x (left top of crop rectangle)")
	yConfig = flag.String("y", "0", "y (left top of crop rectangle)")

	widthConfig  = flag.String("width", "4032", "width of crop rectangle")
	heightConfig = flag.String("height", "3024", "height of crop rectangle")
)

func main() {
	flag.Parse()

	if *hostConfig == "" || *portConfig == "" || *fileConfig == "" {
		flag.Usage()
		return
	}

	x, err := strconv.Atoi(*xConfig)
	if err != nil {
		log.Println(err)
		return
	}
	y, err := strconv.Atoi(*yConfig)
	if err != nil {
		log.Println(err)
		return
	}
	width, err := strconv.Atoi(*widthConfig)
	if err != nil {
		log.Println(err)
		return
	}
	height, err := strconv.Atoi(*heightConfig)
	if err != nil {
		log.Println(err)
		return
	}

	ifile, err := os.Open(*fileConfig)
	if err != nil {
		log.Println(err)
		return
	}
	imgIn, _, err := image.Decode(ifile)
	if err != nil {
		log.Println(err)
		return
	}

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, imgIn, nil)
	sendS3 := buf.Bytes()

	url := fmt.Sprintf("%s:%s", *hostConfig, *portConfig)

	// Set up a connection to the server.
	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	c := engine.NewSTYoloClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	channel, err := c.ConfigUpdater(context.Background())
	channel.Send(&engine.Response{Message: "Channel opened!"})
	channel.Send(&engine.Response{Message: "Channel opened!"})
	cfg := &engine.Config{}
	/*
		//Uncomment second to start test of hot reload and copy right uid
		ans, eerr := c.SetConfig(ctx, &engine.Config{Uid: "b0898235-589b-42f7-b7ee-7a978fd4d944", DetectionLines: []*engine.DetectionLine{&engine.DetectionLine{Id: 2, Begin: &engine.Point{X: 1, Y: 1}, End: &engine.Point{X: 416, Y: 416}}}})
		fmt.Println(ans, eerr)
		return

		//Uncomment first to start test of hot rload
		for {
			cfg, err := channel.Recv()
			fmt.Println(cfg, err)
		}
	*/
	cfg, err = channel.Recv()
	cfg.DetectionLines = []*engine.DetectionLine{&engine.DetectionLine{Id: 1, Begin: &engine.Point{X: 1, Y: 1}, End: &engine.Point{X: 416, Y: 416}}}
	resp, _ := c.SetConfig(ctx, cfg)
	fmt.Println(resp)
	defer cancel()
	r, err := c.SendDetection(
		ctx,
		&engine.CamInfo{
			CamId:     cfg.GetUid(),
			Timestamp: time.Now().Unix(),
			Image:     sendS3,
			Detection: &engine.Detection{
				XLeft:  int32(x),
				YTop:   int32(y),
				Width:  int32(width),
				Height: int32(height),
				LineId: 1,
			},
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	if len(r.GetError()) != 0 {
		log.Println(r.GetError())
		return
	}
	if len(r.GetWarning()) != 0 {
		log.Println("Warn:", r.GetWarning())
	}
	c.SetConfig(ctx, cfg)
	log.Println("Answer:", r.GetMessage())
}

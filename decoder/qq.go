package decoder

import (
  "io/ioutil"
  "os"
  "path"
  "path/filepath"
  "strings"
  "github.com/wsdbd/qn-decoder/logger"
)

var (
  seedMap [][]uint = [][]uint{
    []uint{0x4a, 0xd6, 0xca, 0x90, 0x67, 0xf7, 0x52},
    []uint{0x5e, 0x95, 0x23, 0x9f, 0x13, 0x11, 0x7e},
    []uint{0x47, 0x74, 0x3d, 0x90, 0xaa, 0x3f, 0x51},
    []uint{0xc6, 0x09, 0xd5, 0x9f, 0xfa, 0x66, 0xf9},
    []uint{0xf3, 0xd6, 0xa1, 0x90, 0xa0, 0xf7, 0xf0},
    []uint{0x1d, 0x95, 0xde, 0x9f, 0x84, 0x11, 0xf4},
    []uint{0x0e, 0x74, 0xbb, 0x90, 0xbc, 0x3f, 0x92},
    []uint{0x00, 0x09, 0x5b, 0x9f, 0x62, 0x66, 0xa1},
  }

  x int64 = -1
  y int64 = 8
  dx int64 = 1
  index int64 = -1
)

func nextMask() uint {
  index++

  var ret uint = 0

  if x < 0 {
    dx = 1
    y = (8 - y) % 8
    // ret = (8 - y) % 8
    ret = 0xc3
  } else if x > 6 {
    dx = -1
    y = 7 - y
    ret = 0xd8
  } else {
    ret = seedMap[y][x]
  }

  x += dx
  if index == 0x8000 || (index > 0x8000 && (index + 1) % 0x8000 == 0) {
    return nextMask()
  }

  return ret
}

func DecodeQMC(filePath string, outputFolder string) {
  x = -1
  y = 8
  dx = 1
  index = -1

  buf, err := ioutil.ReadFile(filePath)
  if err != nil {
    logger.Println(err)
    return
  }

  newData := make([]byte, len(buf))

  for i, b := range buf {
    newData[i] = byte(nextMask() ^ uint(b))
  }

  extname := filepath.Ext(filePath)
  basename := filepath.Base(filePath)
  filename := strings.TrimSuffix(basename, extname)

  format := "mp3"
  if extname == ".qmcflac" {
    format = "flac"
  }

  newFilename := filename + "." + format

  outPath := path.Join(outputFolder, newFilename)
  of, err := os.Create(outPath)
  defer of.Close()
  _, err = of.Write([]byte(newData))
  of.Sync()

  logger.Println(basename, "->", newFilename)
}

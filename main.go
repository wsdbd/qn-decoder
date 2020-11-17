package main

import (
  "flag"
  "path/filepath"
  "os"
  "github.com/wsdbd/qn-decoder/logger"
  "github.com/wsdbd/qn-decoder/decoder"
)

func main() {
  var inputPath string
  var outputPath string
  flag.StringVar(&inputPath, "i", ".", "输入文件夹")
  flag.StringVar(&outputPath, "o", "output", "输出文件夹")

  flag.Parse()
  inputPath, err := filepath.Abs(inputPath)
  if err != nil {
    logger.Println("输入文件路径错误！")
    return
  }
  outputPath, err = filepath.Abs(outputPath)
  if err != nil {
    logger.Println("输出文件路径错误！")
    return
  }

  _ = os.Mkdir(outputPath, os.ModePerm)

  err = filepath.Walk(inputPath,
      func(path string, info os.FileInfo, err error) error {
      if err != nil {
          return err
      }

      extname := filepath.Ext(path)
      if extname == ".ncm" {
        decoder.DecodeNCM(path, outputPath)
      } else if extname == ".qmcflac" || extname == ".qmc0" {
        decoder.DecodeQMC(path, outputPath)
      }

      return nil
  })
  if err != nil {
      logger.Println(err)
  }
}

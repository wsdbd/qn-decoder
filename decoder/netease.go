package decoder

import (
  "strconv"
  "os"
  "fmt"
  "path"
  "path/filepath"
  "strings"
  "encoding/binary"
  "encoding/json"
  "crypto/aes"
  "encoding/base64"
  "reflect"

  "github.com/wsdbd/qn-decoder/logger"
  "github.com/bogem/id3v2"
)

var (
  corekey string = "687A4852416D736F356B496E62617857"
  metaKey string = "2331346C6A6B5F215C5D2630553C2728"
  magicHeader string = "4354454e4644414d"
)


func DecodeNCM(filePath string, outputFolder string) {
  f, err := os.Open(filePath)
  if err != nil {
    logger.Println(err)
  }
  defer f.Close()

  header := make([]byte, 8)
  _, err = f.Read(header)
  if err != nil {
    logger.Println(err)
  }

  if string(header) != string(Unhexlify("4354454e4644414d")) {
    logger.Println("不是正常的ncm格式！")
    return
  }

  _, err = f.Seek(2, 1)
  if err != nil {
    logger.Println(err)
  }

  keyLengthBytes := make([]byte, 4)
  f.Read(keyLengthBytes)
  keyLength := binary.LittleEndian.Uint16(keyLengthBytes)
  keyData := make([]byte, keyLength)
  f.Read(keyData)
  newKeyData := make([]byte, keyLength)
  for i, b := range keyData {
    newKeyData[i] = b ^ 0x64
  }

  decryptedKeyData := aesDecryptECB(Unhexlify(corekey), newKeyData)
  newDecryptKey := decryptedKeyData[17:]
  newKeyLength := len(newDecryptKey)

  s := make([]byte, 256)
  for i := 0; i < 256; i++ {
    s[i] = byte(i)
  }
  j := 0

  for i := 0; i < 256; i++ {
    j = (j + int(s[i]) + int(newDecryptKey[i%newKeyLength])) & 0xFF
    s[i], s[j] = s[j], s[i]
  }

  // identifier := ""
  metaLengthBytes := make([]byte, 4)
  f.Read(metaLengthBytes)
  metaLength := binary.LittleEndian.Uint16(metaLengthBytes)
  metaDataMap := make(map[string]interface{})

  info, err := os.Stat(filePath)
  if err != nil {
    logger.Println(err)
    return
  }

  format := "mp3"

  if metaLength > 0 {
    metaData := make([]byte, metaLength)
    f.Read(metaData)
    newMetaData := make([]byte, metaLength)
    for i, b := range metaData {
      newMetaData[i] = b ^ 0x63
    }

    // identifier = string(newMetaData)
    realMeataData, err := base64.StdEncoding.DecodeString(string(newMetaData[22:]))
    if err != nil {
      logger.Println(err)
    }
    decodeMetaData := aesDecryptECB(Unhexlify(metaKey), realMeataData)
    json.Unmarshal(decodeMetaData[6:], &metaDataMap)
  } else {
    if int64(info.Size()) > int64(1024 * 1024 * 16) {
      format = "flac"
    }
    metaDataMap = map[string]interface{}{
      "format": format,
    }
  }

  f.Seek(5, 1)
  imageSpaceBytes := make([]byte, 4)
  f.Read(imageSpaceBytes)
  imageSpace := binary.LittleEndian.Uint32(imageSpaceBytes)
  imageSizeBytes := make([]byte, 4)
  f.Read(imageSizeBytes)
  imageSize := binary.LittleEndian.Uint32(imageSizeBytes)

  imageDataBytes := make([]byte, imageSize)
  if imageSize > 0 {
    f.Read(imageDataBytes)
  }

  pos, err := f.Seek(int64(imageSpace - imageSize), 1)
  if err != nil {
    logger.Println(err)
    return
  }

  dataLen := info.Size() - pos
  data := make([]byte, dataLen)
  f.Read(data)

  stream := make([]byte, 256)
  for i := 0; i < 256; i++ {
    j := (int(s[i]) + int(s[(i + int(s[i])) & 0xFF])) & 0xFF
    stream[i] = s[j]
  }

  newStream := make([]byte, 0)
  for i := 0; i < len(data); i++ {
    v := stream[(i+1)%256]
    newStream = append(newStream, v)
  }

  newData := strxor(string(data), string(newStream))

  extname := filepath.Ext(filePath)
  basename := filepath.Base(filePath)
  filename := strings.TrimSuffix(basename, extname)

  newFilename := filename + "." + format

  outPath := path.Join(outputFolder, newFilename)
  of, _ := os.Create(outPath)
  defer of.Close()
  _, err = of.Write([]byte(newData))
  if err != nil {
    logger.Println(err)
    return
  }
  of.Sync()

  if format == "mp3" {
    mp3File, err := id3v2.Open(outPath, id3v2.Options{Parse: false})
    defer mp3File.Close()

    if err != nil {
      logger.Println(err)
    }

    mp3File.SetDefaultEncoding(id3v2.EncodingUTF8)

    if artists, ok := metaDataMap["artist"]; ok {
      tp := reflect.TypeOf(artists)
      nameArr := make([]string, 0)
      switch tp.Kind() {
      case reflect.Slice, reflect.Array:
        items := reflect.ValueOf(artists)
        for i := 0; i < items.Len(); i++ {
          item := items.Index(i).Interface()
          tp1 := reflect.TypeOf(item)
          switch tp1.Kind() {
          case reflect.Slice, reflect.Array:
            values := reflect.ValueOf(item)
            if values.Len() > 0 {
              nameArr = append(nameArr, values.Index(0).Interface().(string))
            }
          }
        }
      }
      artistName := strings.Join(nameArr, "/")
      mp3File.SetArtist(artistName)
    }
    mp3File.SetTitle(fmt.Sprintf("%v", metaDataMap["musicName"]))
    mp3File.SetAlbum(fmt.Sprintf("%v", metaDataMap["album"]))

    if len(imageDataBytes) > 0 {
      pic := id3v2.PictureFrame{
    		Encoding:    id3v2.EncodingISO,
    		MimeType:    "image/jpeg",
    		PictureType: id3v2.PTFrontCover,
    		Description: "Front cover",
    		Picture:     imageDataBytes,
    	}
    	mp3File.AddAttachedPicture(pic)
    }

    if err = mp3File.Save(); err != nil {
      logger.Println(err)
    }

  } else {

  }


  logger.Println(basename, "->", newFilename)
}

// func NewImageFrame(ft idv23.FrameType, mime_type string, image_data []byte) *idv23.ImageFrame {
//     data_frame := idv23.NewDataFrame(ft, image_data)
//     data_frame.size += uint32(1)
//
//     // ID3 standard says the string has to be null-terminated.
//     nullTermBytes := append(image_data, 0x00)
//
//     image_frame := &idv23.ImageFrame{
//         DataFrame:   *data_frame, // DataFrame header
//         pictureType: byte(0x03),  // Image Type, in this case Front Cover (http://id3.org/id3v2.3.0#Attached_picture)
//         description: string(nullTermBytes),
//     }
//     image_frame.SetEncoding("UTF-8")
//     image_frame.SetMIMEType(mime_type)
//     return image_frame
// }
//

func strxor(s1, s2 string) string {
  if len(s1) != len(s2) {
    panic("strXor called with two strings of different length\n")
  }
  n := len(s1)
  b := make([]byte, n)
  for i := 0; i < n; i++ {
    b[i] = s1[i] ^ s2[i]
  }
  return string(b)
}

func Unpad(data []byte, blockSize uint) ([]byte, error) {
	if blockSize < 1 {
		return nil, fmt.Errorf("Block size looks wrong")
	}

	if uint(len(data))%blockSize != 0 {
		return nil, fmt.Errorf("Data isn't aligned to blockSize")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("Data is empty")
	}

	paddingLength := int(data[len(data)-1])
	for _, el := range data[len(data)-paddingLength:] {
		if el != byte(paddingLength) {
			return nil, fmt.Errorf("Padding had malformed entries. Have '%x', expected '%x'", paddingLength, el)
		}
	}

	return data[:len(data)-paddingLength], nil
}


func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
}

func aesDecryptECB(key []byte, encrypted []byte) ([]byte) {
  cipher, _ := aes.NewCipher(generateKey(key))
  decrypted := make([]byte, len(encrypted))

  for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
    cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
  }

  trim := 0
  if len(decrypted) > 0 {
    trim = len(decrypted) - int(decrypted[len(decrypted)-1])
  }

  return decrypted[:trim]
}

func Unhexlify(str string) []byte {
    res := make([]byte, 0)
    for i := 0; i < len(str); i+=2 {
        x, _ := strconv.ParseInt(str[i:i+2], 16, 32)
        res = append(res, byte(x))
    }
    return res
}

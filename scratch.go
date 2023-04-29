package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pierrec/lz4"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Usage = "Tar and compress files and directories using LZ4"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "output file name",
			Value:   "output.tar.lz4",
		},
	}
	app.Action = func(c *cli.Context) error {
		// Create the output file
		outFile, err := os.Create(c.String("output"))
		if err != nil {
			return err
		}
		defer outFile.Close()

		// Create a new tar writer
		tarWriter := tar.NewWriter(outFile)
		defer tarWriter.Close()

		// Loop over input files and directories
		for _, input := range c.Args().Slice() {
			err = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Create a new tar header
				header, err := tar.FileInfoHeader(info, info.Name())
				if err != nil {
					return err
				}
				header.Name = path

				// Write the header to the tar archive
				err = tarWriter.WriteHeader(header)
				if err != nil {
					return err
				}

				// Write the file data to the tar archive
				if !info.IsDir() {
					file, err := os.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()

					_, err = io.Copy(tarWriter, file)
					if err != nil {
						return err
					}
				}

				return nil
			})
			if err != nil {
				return err
			}
		}

		// Create a new lz4 writer
		lz4Writer := lz4.NewWriter(outFile)
		defer lz4Writer.Close()

		// Compress the tar archive using lz4
		_, err = io.Copy(lz4Writer, outFile)
		if err != nil {
			return err
		}

		// Delete the tar archive
		// err = os.Remove(c.String("output") + ".tar")
		// if err != nil {
		// 	return err
		// }

		fmt.Println("Tar archive created and compressed successfully!")

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

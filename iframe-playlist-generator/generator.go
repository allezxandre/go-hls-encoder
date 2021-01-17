package iframe_playlist_generator

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"fmt"
	"github.com/grafov/m3u8"
)

func EnrichPlaylist(dirMaster, masterFilename, dirInfo, infoFilename, newName string) (string, error) {
	// Open Playlist to "enrich"
	inFileFullPath := filepath.Join(dirMaster, masterFilename)
	p, _, t, err := variantsFromMaster(inFileFullPath)
	if err != nil {
		return "", err
	}
	if t != m3u8.MASTER {
		log.Println("Cannot Enrich playlist", masterFilename, "as it is not a Master Playlist. Type:", t)
		return "", nil
	}
	pMaster := p.(*m3u8.MasterPlaylist)
	// Open other playlist
	inFileFullPath = filepath.Join(dirInfo, infoFilename)
	p_, _, t, err := variantsFromMaster(inFileFullPath)
	if err != nil {
		return "", err
	}
	if t != m3u8.MASTER {
		log.Println("Cannot Enrich playlist", masterFilename, "as it is not a Master Playlist. Type:", t)
		return "", nil
	}
	pInfo := p_.(*m3u8.MasterPlaylist)

	// Use info from pInfo to enrich pMaster
	for _, v := range pMaster.Variants {
		updateVariant(pInfo, v)
	}

	// Write playlist
	return writePlaylistToFile(pMaster, dirMaster, newName)
}

func updateVariant(playlistWithInfo *m3u8.MasterPlaylist, v *m3u8.Variant) {
	for _, v_ := range playlistWithInfo.Variants {
		if v_.URI == v.URI {
			// Update v with v_
			v.VariantParams.Bandwidth = v_.VariantParams.Bandwidth
			if len(v_.VariantParams.Codecs) > 0 {
				v.VariantParams.Codecs = v_.VariantParams.Codecs
			}
			break
		}
	}
}

func GeneratePlaylist(dir, inFile string) error {
	// Retrieve variants
	inFileFullPath := filepath.Join(dir, inFile)
	_, variants, t, err := variantsFromMaster(inFileFullPath)
	if err != nil {
		return err
	}

	// File for writing
	var f *os.File = nil
	if t == m3u8.MASTER {
		f, err = os.OpenFile(inFileFullPath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	// Fill variants chunks
	fillVariants(dir, variants...)

	// Generate and write i-frame only playlists
	for _, variant := range variants {
		// Generate playlist
		iframePlaylist, err := iframePlaylistForVariant(dir, variant)
		if err != nil {
			log.Println("Cannot generate I-FRAMES-ONLY playlist for variant \""+variant.URI+
				"\"... Carrying on with the others anyway. \n\tError:", err)
			continue
		}
		log.Println("DEBUG: Writing playlist")
		// Write to new file
		iframePlaylist.TargetDuration -= 1
		iframeFilename, err := writePlaylistToFile(iframePlaylist, dir, iframeOnlyFilename(variant.URI))
		if err != nil {
			log.Println("Cannot write I-FRAMES-ONLY playlist to file \""+variant.URI+
				"\"... Carrying on with the others anyway. \n\tError:", err)
			continue
		}
		log.Println("DEBUG: Appending playlists to master?")
		// Append to master, if master
		if t == m3u8.MASTER {
			log.Println("DEBUG:        -> yes")
			_, err := f.WriteString(fmt.Sprintf(
				"#EXT-X-I-FRAME-STREAM-INF:BANDWIDTH=%d,URI=\"%v\"\n",
				int(variant.Bandwidth)/10, iframeFilename))
			// TODO: To determine bandwidth, instead of dividing by 10, sum the size of all iFrames and divide by duration
			if err != nil {
				log.Println("Error writing to master:", err)
			}
		} else {
			log.Println("DEBUG:        -> NO??? Something's wrong dude...")
		}
	}

	log.Println("DEBUG: I-FRAME-GENERATION Done")
	return nil
}

// variantsFromMaster Returns a slice of variants to use
// contained in an m3u8 file. Automatically checks the type
// of the playlist (Master or Media playlist).
//
func variantsFromMaster(playlistPath string) (m3u8.Playlist, []*m3u8.Variant, m3u8.ListType, error) {
	f, err := os.Open(playlistPath)
	if err != nil {
		return nil, []*m3u8.Variant{}, 0, err
	}
	defer f.Close()
	p, t, err := m3u8.DecodeFrom(f, false)
	if err != nil {
		return nil, []*m3u8.Variant{}, 0, err
	}
	switch t {
	case m3u8.MASTER:
		variants := p.(*m3u8.MasterPlaylist).Variants
		return p, variants, t, nil
	case m3u8.MEDIA:
		p := p.(*m3u8.MediaPlaylist)
		variant := m3u8.Variant{
			URI:       playlistPath,
			Chunklist: p,
		}
		return nil, []*m3u8.Variant{&variant}, t, nil
	default:
		err := errors.New("assertion error: unknown mediaplaylist type")
		return nil, []*m3u8.Variant{}, t, err
	}
}

// fillVariants Reads files to fill variants chunklists
func fillVariants(dir string, variants ...*m3u8.Variant) {
	for _, v := range variants {
		uri := filepath.Join(dir, v.URI)
		p, _ := m3u8.NewMediaPlaylist(0, 1)
		v.Chunklist = p
		f, err := os.Open(uri)
		if err != nil {
			log.Println("Cannot read variant at URI \"" + v.URI + "\". Skipping variant...")
			continue
		}
		err = v.Chunklist.DecodeFrom(f, true)
		if err != nil {
			log.Println("Cannot decode variant at URI \"" + v.URI + "\". Skipping variant...")
			continue
		}
		f.Close()
	}
}

// iframePlaylistForVariant Generates an I-FRAMES-ONLY media playlist
// for the provided variant.
func iframePlaylistForVariant(dir string, variant *m3u8.Variant) (*m3u8.MediaPlaylist, error) {
	if variant.Chunklist == nil {
		return nil, errors.New("`nil` chunklist for variant \"" + variant.URI + "\"")
	}

	// Loop through segments of variant to find key frames
	var entries []*IFrameEntry
	nbSegmts := len(variant.Chunklist.Segments)
	initFilename := ""
	var initSize uint = 0
	for i, segment := range variant.Chunklist.Segments {
		if segment == nil {
			break // TODO: fix this library
		}
		if segment.Map != nil {
			initFilename = filepath.Join(dir, segment.Map.URI)
			fi, _ := os.Stat(initFilename)
			initSize = uint(fi.Size())
		}
		entriesPartial, err := iframeEntryForSegment(initFilename, initSize, filepath.Join(dir, segment.URI))
		if err != nil {
			log.Println("DEBUG: Error running iframeEntryForSegment on", filepath.Join(dir, segment.URI))
			return nil, err
		}
		entries = append(entries, entriesPartial...)
		fmt.Printf("DEBUG: EXT-I-Frame Progress: %d/%d\n", i, nbSegmts)
	}

	// Generate playlist from entries
	log.Println("DEBUG: Generating playlist")
	p, _ := m3u8.NewMediaPlaylist(0, uint(len(entries)))
	p.SetIframeOnly()
	p.TargetDuration = variant.Chunklist.TargetDuration
	for _, entry := range entries {
		p.Append(entry.SegmentURI, entry.Duration, "")
		p.SetRange(int64(entry.PacketSize), int64(entry.PacketPosition))
	}

	return p, nil
}

type IFrameEntry struct {
	SegmentURI     string
	PacketPosition uint
	PacketSize     uint
	Duration       float64
}

// iframeEntryForSegment Looks for all IFrames packets position and size/duration.
func iframeEntryForSegment(initURI string, initSize uint, segmentURI string) ([]*IFrameEntry, error) {
	packets, err := probePackets(initURI, segmentURI)
	if err != nil {
		return nil, err
	}

	var entries []*IFrameEntry
	var lastEntry *IFrameEntry = nil

	nbPkts := len(packets)
	for i, p := range packets {
		if p.isFromKeyFrame() {
			// Save last entry
			if lastEntry != nil {
				entries = append(entries, lastEntry)
			}
			// Compute packet size
			size := p.Size
			if i < nbPkts-1 {
				pNext := packets[i+1]
				size = pNext.Pos - p.Pos
			}
			size += 188 // I have no idea why
			// Generate entry
			lastEntry = &IFrameEntry{
				SegmentURI:     filepath.Base(segmentURI),
				PacketPosition: p.Pos - initSize,
				PacketSize:     size,
				Duration:       p.DurationTime,
			} // Duration added when next keyframe found
		} else {
			if lastEntry != nil {
				lastEntry.Duration += p.DurationTime
			}
		}
	}

	// Save last entry
	if lastEntry != nil {
		entries = append(entries, lastEntry)
	} else {
		log.Println("WARNING: Segment", segmentURI, "has no key frame.")
	}
	return entries, nil
}

func iframeOnlyFilename(originalName string) (newName string) {
	extName := filepath.Ext(originalName)
	bName := originalName[:len(originalName)-len(extName)]
	newName = bName + "_I-FRAME-ONLY" + extName
	return
}

// writePlaylistToFile Writes the playlist to a file
func writePlaylistToFile(p m3u8.Playlist, dir string, playlistFilename string) (string, error) {
	playlistFullPath := filepath.Join(dir, playlistFilename)
	f, err := os.OpenFile(playlistFullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", err // FIXME
	}

	_, err = f.Write(p.Encode().Bytes())
	if err != nil {
		return "", err // FIXME
	}

	return playlistFilename, nil
}

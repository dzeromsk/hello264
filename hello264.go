package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"unsafe"

	"github.com/Eyevinn/mp4ff/bits"
)

const (
	Width  = 1280
	Height = 720
)

// YUV ffmpeg data
type frame struct {
	Y  [Height][Width]byte
	Cb [Height / 2][Width / 2]byte
	Cr [Height / 2][Width / 2]byte
}

var nal = []byte{0x00, 0x00, 0x00, 0x01}

// 7.3.2.1.1 Sequence parameter set data syntax
var sps = NewBitBuffer(
	Write(1, 0),         // forbidden_zero_bit
	Write(2, 3),         // nal_ref_idc
	Write(5, 7),         // nal_unit_type
	Write(8, 66),        // profile_idc
	Write(1, 0),         // constraint_set0_flag
	Write(1, 0),         // constraint_set1_flag
	Write(1, 0),         // constraint_set2_flag
	Write(1, 0),         // constraint_set3_flag
	Write(4, 0),         // reserved_zero_4bits
	Write(8, 10),        // level_idc
	Write(V, 0),         // seq_parameter_set_id
	Write(V, 0),         // log2_max_frame_num_minus4
	Write(V, 0),         // pic_order_cnt_type
	Write(V, 0),         // log2_max_pic_order_cnt_lsb_minus4
	Write(V, 0),         // num_ref_frames
	Write(1, 0),         // gaps_in_frame_num_value_allowed_flag
	Write(V, b(Width)),  // pic_width_in_mbs_minus_1
	Write(V, b(Height)), // pic_height_in_map_units_minus_1
	Write(1, 1),         // frame_mbs_only_flag
	Write(1, 0),         // direct_8x8_inference_flag
	Write(1, 0),         // frame_cropping_flag
	Write(1, 0),         // vui_prameters_present_flag
	Write(1, 1),         // rbsp_stop_one_bit
)

func b(x uint) uint {
	return x/16 - 1
}

// 7.3.2.2 Picture parameter set RBSP syntax
var pps = NewBitBuffer(
	Write(1, 0), // forbidden_zero_bit
	Write(2, 3), // nal_ref_idc
	Write(5, 8), // nal_unit_type
	Write(V, 0), // pic_parameter_set_id
	Write(V, 0), // seq_parameter_set_id
	Write(1, 0), // entropy_coding_mode_flag
	Write(1, 0), // bottom_field_pic_order_in_frame_present_flag
	Write(V, 0), // num_slice_groups_minus1
	Write(V, 0), // num_ref_idx_l0_default_active_minus1
	Write(V, 0), // num_ref_idx_l1_default_active_minus1
	Write(1, 0), // weighted_pred_flag
	Write(2, 0), // weighted_bipred_idc
	Write(V, 0), // pic_init_qp_minus26
	Write(V, 0), // pic_init_qs_minus26
	Write(V, 0), // chroma_qp_index_offset
	Write(1, 0), // deblocking_filter_control_present_flag
	Write(1, 0), // constrained_intra_pred_flag
	Write(1, 0), // redundant_pic_cnt_present_flag
	Write(V, 0), // second_chroma_qp_index_offset
)

// 7.3.3 Slice header syntax
var slicemb = NewBitBuffer(
	Write(1, 0), // forbidden_zero_bit
	Write(2, 0), // nal_ref_idc
	Write(5, 5), // nal_unit_type
	Write(V, 0), // first_mb_in_slice
	Write(V, 7), // slice_type
	Write(V, 0), // pic_parameter_set_id
	Write(4, 0), // frame_num
	Write(V, 0), // idr_pic_id
	Write(4, 0), // pic_order_cnt_lsb
	Write(V, 0), // slice_qp_delta

	// Fused with first macroblock type to avoid alignment issues
	Write(V, 25), // mb_type I_PCM
)

// 7.3.5 Macroblock layer syntax
var mb = NewBitBuffer(
	Write(V, 25), // mb_type I_PCM
)

var stop = NewBitBuffer(
	Write(1, 1),
)

func NewBitBuffer(writes ...WriteFn) []byte {
	var b bytes.Buffer
	w := bits.NewEBSPWriter(&b)
	for _, f := range writes {
		f(w)
	}
	w.StuffByteWithZeros()
	return b.Bytes()
}

type WriteFn func(*bits.EBSPWriter)

const V = math.MaxInt

func Write(n int, v uint) WriteFn {
	return func(w *bits.EBSPWriter) {
		switch n {
		case V:
			w.WriteExpGolomb(v)
		default:
			w.Write(v, n)
		}
	}
}

// Emulation Prevention Writer
type epwriter struct {
	io.Writer
	p uint16
}

func newepwriter(w io.Writer) epwriter {
	return epwriter{
		Writer: w,
		p:      0xffff,
	}
}

func (b *epwriter) WriteByte(c byte) error {
	if b.p == 0x0000 && c&0xfc == 0x00 {
		b.Writer.Write([]byte{0x03})
		b.p |= 0x03
	}
	b.p <<= 8
	b.p |= uint16(c)
	_, err := b.Writer.Write([]byte{c})
	return err
}

func bytesArray(f *frame) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(f)), unsafe.Sizeof(*f))
}

func macroBlock(w io.Writer, f *frame, x int, y int) {
	b := newepwriter(w)

	bx, by := x*16, y*16
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			b.WriteByte(f.Y[bx+i][by+j])
		}
	}

	bx, by = x*8, y*8
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			b.WriteByte(f.Cb[bx+i][by+j])
		}
	}

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			b.WriteByte(f.Cr[bx+i][by+j])
		}
	}
}

var (
	stdin  = os.Stdin
	stdout = bufio.NewWriter(os.Stdout)
	stderr = os.Stderr
)

func main() {
	stdout.Write(nal) // 0x00000001
	stdout.Write(sps) // 0x6742000af80a00b620

	stdout.Write(nal) // 0x00000001
	stdout.Write(pps) // 0x68ce3880

	var f frame
	for {
		_, err := stdin.Read(bytesArray(&f))
		if err == io.EOF {
			break
		}

		stdout.Write(nal)     // 0x00000001
		stdout.Write(slicemb) // 0x05888421a0

		for i := 0; i < Height/16; i++ {
			for j := 0; j < Width/16; j++ {
				if i != 0 || j != 0 {
					stdout.Write(mb) // 0x0d00
				}
				macroBlock(stdout, &f, i, j)
			}
		}

		stdout.Write(stop) // 0x80
	}

	stdout.Flush()
	debugdump()
}

func debugdump() {
	fmt.Fprintln(stderr, "nal:", hex.EncodeToString(nal))
	fmt.Fprintln(stderr, "sps:", hex.EncodeToString(sps))
	fmt.Fprintln(stderr, "pps:", hex.EncodeToString(pps))
	fmt.Fprintln(stderr, "slice:", hex.EncodeToString(slicemb))
	fmt.Fprintln(stderr, "mb:", hex.EncodeToString(mb))
	fmt.Fprintln(stderr, "stop:", hex.EncodeToString(stop))
}

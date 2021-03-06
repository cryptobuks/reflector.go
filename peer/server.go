package peer

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/lbryio/reflector.go/store"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultPort    = 3333
	LbrycrdAddress = "bJxKvpD96kaJLriqVajZ7SaQTsWWyrGQct"
)

type Server struct {
	store store.BlobStore
}

func NewServer(store store.BlobStore) *Server {
	return &Server{
		store: store,
	}
}

func (s *Server) ListenAndServe(address string) error {
	log.Println("Listening on " + address)
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error(err)
		} else {
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	timeoutDuration := 5 * time.Second

	for {
		var request []byte
		var response []byte
		var err error

		conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		request, err = readNextRequest(conn)
		if err != nil {
			if err != io.EOF {
				log.Errorln(err)
			}
			return
		}
		conn.SetReadDeadline(time.Time{})

		if strings.Contains(string(request), `"requested_blobs"`) {
			log.Debugln("received availability request")
			response, err = s.handleAvailabilityRequest(request)
		} else if strings.Contains(string(request), `"blob_data_payment_rate"`) {
			log.Debugln("received rate negotiation request")
			response, err = s.handlePaymentRateNegotiation(request)
		} else if strings.Contains(string(request), `"requested_blob"`) {
			log.Debugln("received blob request")
			response, err = s.handleBlobRequest(request)
		} else {
			log.Errorln("invalid request")
			spew.Dump(request)
			return
		}
		if err != nil {
			log.Error(err)
			return
		}

		n, err := conn.Write(response)
		if err != nil {
			log.Errorln(err)
			return
		} else if n != len(response) {
			log.Errorln(io.ErrShortWrite)
			return
		}
	}
}

func (s *Server) handleAvailabilityRequest(data []byte) ([]byte, error) {
	var request availabilityRequest
	err := json.Unmarshal(data, &request)
	if err != nil {
		return []byte{}, err
	}

	availableBlobs := []string{}
	for _, blobHash := range request.RequestedBlobs {
		exists, err := s.store.Has(blobHash)
		if err != nil {
			return []byte{}, err
		}
		if exists {
			availableBlobs = append(availableBlobs, blobHash)
		}
	}

	return json.Marshal(availabilityResponse{LbrycrdAddress: LbrycrdAddress, AvailableBlobs: availableBlobs})
}

func (s *Server) handlePaymentRateNegotiation(data []byte) ([]byte, error) {
	var request paymentRateRequest
	err := json.Unmarshal(data, &request)
	if err != nil {
		return []byte{}, err
	}

	offerReply := paymentRateAccepted
	if request.BlobDataPaymentRate < 0 {
		offerReply = paymentRateTooLow
	}

	return json.Marshal(paymentRateResponse{BlobDataPaymentRate: offerReply})
}

func (s *Server) handleBlobRequest(data []byte) ([]byte, error) {
	var request blobRequest
	err := json.Unmarshal(data, &request)
	if err != nil {
		return []byte{}, err
	}

	log.Println("Sending blob " + request.RequestedBlob[:8])

	blob, err := s.store.Get(request.RequestedBlob)
	if err != nil {
		return []byte{}, err
	}

	response, err := json.Marshal(blobResponse{IncomingBlob: incomingBlob{
		BlobHash: getBlobHash(blob),
		Length:   len(blob),
	}})
	if err != nil {
		return []byte{}, err
	}

	return append(response, blob...), nil
}

func readNextRequest(conn net.Conn) ([]byte, error) {
	request := make([]byte, 0)
	eof := false
	buf := bufio.NewReader(conn)

	for {
		chunk, err := buf.ReadBytes('}')
		if err != nil {
			if err != io.EOF {
				log.Errorln("read error:", err)
				return request, err
			}
			eof = true
		}

		//log.Debugln("got", len(chunk), "bytes.")
		//spew.Dump(chunk)

		if len(chunk) > 0 {
			request = append(request, chunk...)

			if len(request) > maxRequestSize {
				return request, errRequestTooLarge
			}

			// yes, this is how the peer protocol knows when the request finishes
			if isValidJSON(request) {
				break
			}
		}

		if eof {
			break
		}
	}

	//log.Debugln("total size:", len(request))
	//if len(request) > 0 {
	//	spew.Dump(request)
	//}

	if len(request) == 0 && eof {
		return []byte{}, io.EOF
	}

	return request, nil
}

func isValidJSON(b []byte) bool {
	var r json.RawMessage
	return json.Unmarshal(b, &r) == nil
}

func getBlobHash(blob []byte) string {
	hashBytes := sha512.Sum384(blob)
	return hex.EncodeToString(hashBytes[:])
}

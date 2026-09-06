package fireblazer

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func EnumerateServiceAccounts(projectNum string, workerCount int, updateCh chan ScanUpdate, rawKey string) ([]string, error) {
	var mu sync.Mutex
	var foundProducts []string

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(workerCount)

	for _, product := range p4saProducts {
		product := product // do NOT question my sanity
		g.Go(func() error {
			email := fmt.Sprintf("service-%s@gcp-sa-%s.iam.gserviceaccount.com", projectNum, product)
			url := fmt.Sprintf("https://iam.googleapis.com/v1/projects/%s/serviceAccounts/%s", projectNum, email)

			for retries := 0; retries < 3; retries++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					return nil
				}

				// this is actually an unauthed project thing LMAO you could use this for any project ID with no key at all
				resp, err := ReqHeaderOnly(*req, "", false)
				if err != nil {
					return nil
				}

				statusCode := resp.StatusCode
				resp.Body.Close()

				if statusCode == 200 || statusCode == 403 { // lowk i should flag an extra thing if it *is* 200...
					mu.Lock()
					foundProducts = append(foundProducts, product)
					mu.Unlock()
					if updateCh != nil {
						updateCh <- ScanUpdate{Key: rawKey, WasFound: true, ItemCleanName: product}
					}
					return nil
				} else if statusCode == 502 || statusCode == 503 || statusCode == 429 {
					time.Sleep(2 * time.Second) // a 429 should NEVER happen. I mean, NEVER. It's an unauthed resource. Still, I've dealt with too many edgecases with Drive, and want to ensure nothing like that happens again.
					continue                    // never thought i'd need a retry...
				} else {
					if updateCh != nil {
						updateCh <- ScanUpdate{Key: rawKey, WasFound: false, ItemCleanName: product}
					}
					return nil
				}
			}
			if updateCh != nil {
				updateCh <- ScanUpdate{Key: rawKey, WasFound: false, ItemCleanName: product}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return foundProducts, nil
}

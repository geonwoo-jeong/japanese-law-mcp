package hanreipdf

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

const testWorkerBehaviorEnv = "JLMCP_HANREIPDF_TEST_WORKER"

func TestMain(m *testing.M) {
	switch os.Getenv(testWorkerBehaviorEnv) {
	case "hang":
		time.Sleep(time.Hour)
		os.Exit(2)
	case "panic":
		panic("検査用 panic の非公開本文")
	case "hang-after-temp":
		if os.Getenv(privateWorkerEnv) == privateWorkerModePDF {
			os.Exit(runPrivateWorker(os.Stdin, os.Stdout, func(pdfBytes []byte) (workerOutput, error) {
				file, _, err := createWorkerTempPDF(pdfBytes, os.Getenv(privateWorkerTempEnv))
				if err != nil {
					return workerOutput{}, err
				}
				_ = file.Close()
				time.Sleep(time.Hour)
				return workerOutput{}, nil
			}))
		}
		time.Sleep(time.Hour)
		os.Exit(2)
	}
	if handled, code := RunPrivateWorkerIfRequested(
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Getenv(privateWorkerEnv),
	); handled {
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func TestProductionWorkerRunnerUsesSameExecutableProtocol(t *testing.T) {
	output, err := productionWorkerRunner(
		context.Background(),
		syntheticJapanesePDF("平成30(受)10"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if output.TextUnavailable || len(output.Occurrences) != 1 ||
		output.Occurrences[0].DecisionIdentity != "平成30(受)10" {
		t.Fatalf("output=%#v", output)
	}
}

func TestWorkerProcessKillsAndWaitsOnCancellation(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = runWorkerProcess(
		ctx,
		[]byte("%PDF-test"),
		executable,
		[]string{testWorkerBehaviorEnv + "=hang"},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%T %v", err, err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("worker の kill と Wait が期限内に完了しませんでした")
	}
}

func TestProductionWorkerRunnerRemovesTempDirectoryAfterForcedKill(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = runPrivateWorkerProcessWithTempRoot(
		ctx,
		[]byte("%PDF-test"),
		executable,
		root,
		[]string{testWorkerBehaviorEnv + "=hang-after-temp"},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%T %v", err, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("親が child 用 temp directory を回収していません: %#v", entries)
	}
}

func TestWorkerProcessContainsChildPanic(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	_, err = runWorkerProcess(
		context.Background(),
		[]byte("%PDF-test"),
		executable,
		[]string{testWorkerBehaviorEnv + "=panic"},
	)
	var classified workerError
	if !errors.As(err, &classified) || classified.failure != workerFailureProcessingLimit {
		t.Fatalf("error=%T %v", err, err)
	}
}

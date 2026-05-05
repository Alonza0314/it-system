package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"
	"github.com/free-ran-ue/util"
)

type httpSenderMessage struct {
	msgType constant.HttpSenderMessageType

	*model.RequestRunnerHeartbeat
	*model.RequestTestOutput
}

func newHttpSenderMessage(msgType constant.HttpSenderMessageType, heartbeat *model.RequestRunnerHeartbeat, testOutput *model.RequestTestOutput) httpSenderMessage {
	return httpSenderMessage{
		msgType: msgType,

		RequestRunnerHeartbeat: heartbeat,
		RequestTestOutput:      testOutput,
	}
}

type httpSenderServer struct {
	runnerName string

	controllerIP   string
	controllerPort int

	token string

	msgChan       chan httpSenderMessage
	msgChanCtx    context.Context
	msgChanCancel context.CancelFunc

	taskChan chan model.ResponseRunnerHeartbeat

	status constant.RunnerStatus

	*logger.RunnerLogger
}

func newHttpSenderServer(runnerName, controllerIP string, controllerPort, httpSenderChannelSize int, token string, msgChannel chan httpSenderMessage, taskChannel chan model.ResponseRunnerHeartbeat, logger *logger.RunnerLogger) *httpSenderServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &httpSenderServer{
		runnerName: runnerName,

		controllerIP:   controllerIP,
		controllerPort: controllerPort,

		token: token,

		msgChan:       msgChannel,
		msgChanCtx:    ctx,
		msgChanCancel: cancel,

		taskChan: taskChannel,

		status: constant.STATUS_IDLE,

		RunnerLogger: logger,
	}
}

func (s *httpSenderServer) Start() error {
	go func() {
		for {
			select {
			case <-s.msgChanCtx.Done():
				for {
					select {
					case msg := <-s.msgChan:
						s.dispatchMessage(msg)
					default:
						s.HttpLog.Debugln("HttpSenderServer channel drained, exiting.")
						return
					}
				}
			case msg := <-s.msgChan:
				s.dispatchMessage(msg)
			}
		}
	}()

	return nil
}

func (s *httpSenderServer) Stop() error {
	s.msgChanCancel()

	return nil
}

func (s *httpSenderServer) dispatchMessage(msg httpSenderMessage) {
	s.HttpLog.Debugf("Dispatched message of type: %s", msg.msgType)

	switch msg.msgType {
	case constant.MSG_TYPE_HEARTBEAT:
		if s.status == constant.STATUS_IDLE {
			s.HttpLog.Tracef("Runner is idle, processing heartbeat message: %+v", msg.RequestRunnerHeartbeat)
			s.sendHeartbeat(msg.RequestRunnerHeartbeat)
		} else {
			s.HttpLog.Tracef("Runner is running, ignoring heartbeat message: %+v", msg.RequestRunnerHeartbeat)
		}
	case constant.MSG_TYPE_TEST_OUTPUT:
		s.HttpLog.Tracef("Processing test output message: %+v", msg.RequestTestOutput)
		s.sendTestOutput(msg.RequestTestOutput)
	}
}

func (s *httpSenderServer) getHeartbeatUrl() string {
	return fmt.Sprintf(constant.API_RUNNER_HEARTBEAT, s.controllerIP, s.controllerPort)
}

func (s *httpSenderServer) getTestOutputUrl() string {
	return fmt.Sprintf(constant.API_RUNNER_TEST_OUTPUT, s.controllerIP, s.controllerPort)
}

func (s *httpSenderServer) getHeader() map[string]string {
	return map[string]string{
		constant.AUTHENTICATION_HEADER_KEY: fmt.Sprintf(constant.AUTHENTICATION_HEADER_VALUE, s.token),
		constant.USER_HEADER_KEY:           s.runnerName,
	}
}

func (s *httpSenderServer) sendHeartbeat(heartbeat *model.RequestRunnerHeartbeat) {
	data, err := json.Marshal(heartbeat)
	if err != nil {
		s.HttpLog.Errorf("Failed to marshal heartbeat data: %v", err)
		return
	}

	sendSuccess := false
	for t := 0; t < constant.HTTP_RETRY_TIMES && !sendSuccess; t++ {
		s.HttpLog.Debugf("Sending heartbeat to controller (attempt %d/%d)...", t+1, constant.HTTP_RETRY_TIMES)

		response, err := util.SendHttpRequest(s.getHeartbeatUrl(), constant.API_RUNNER_HEARTBEAT_ACTION, s.getHeader(), data)
		if err != nil {
			s.HttpLog.Errorf("Failed to send heartbeat to controller: %v", err)
			continue
		}

		switch response.StatusCode {
		case http.StatusOK:
			sendSuccess = true
			s.setStatusRunning()
			s.HttpLog.Debugf("Controller responded with task for heartbeat, runner status set to running.")

			var heartbeatResponse model.ResponseRunnerHeartbeat
			if err := json.Unmarshal(response.Body, &heartbeatResponse); err != nil {
				s.HttpLog.Errorf("Failed to unmarshal heartbeat response: %v", err)
				continue
			}

			s.taskChan <- heartbeatResponse
		case http.StatusNoContent:
			sendSuccess = true
			s.HttpLog.Debugf("Controller responded with no content for heartbeat.")
		default:
			var message model.ResponseRunnerHeartbeat
			if err := json.Unmarshal(response.Body, &message); err != nil {
				s.HttpLog.Errorf("Failed to unmarshal heartbeat response: %v", err)
				continue
			}

			s.HttpLog.Errorf("Failed to send heartbeat to controller, status code: %d, message: %v", response.StatusCode, message)
			continue
		}
	}

	if !sendSuccess {
		s.HttpLog.Errorf("Failed to send heartbeat to controller after %d attempts.", constant.HTTP_RETRY_TIMES)
		s.HttpLog.Debugf("Heartbeat data that failed to send: %+v", heartbeat)
	}
}

func (s *httpSenderServer) sendTestOutput(testOutput *model.RequestTestOutput) {
	data, err := json.Marshal(testOutput)
	if err != nil {
		s.HttpLog.Errorf("Failed to marshal test output data: %v", err)
		return
	}

	sendSuccess := false
	for t := 0; t < constant.HTTP_RETRY_TIMES && !sendSuccess; t++ {
		s.HttpLog.Debugf("Sending test output to controller (attempt %d/%d)...", t+1, constant.HTTP_RETRY_TIMES)

		response, err := util.SendHttpRequest(s.getTestOutputUrl(), constant.API_RUNNER_TEST_OUTPUT_ACTION, s.getHeader(), data)
		if err != nil {
			s.HttpLog.Errorf("Failed to send test output to controller: %v", err)
			continue
		}

		switch response.StatusCode {
		case http.StatusNoContent:
			sendSuccess = true
			if testOutput.EndFlag == &sendSuccess {
				s.setStatusIdle()
				s.HttpLog.Debugf("Controller responded with no content for test output with end flag true, runner status set to idle.")
			} else {
				s.HttpLog.Debugf("Controller responded with no content for test output.")
			}
		default:
			var message model.ResponseRunnerTestOutput
			if err := json.Unmarshal(response.Body, &message); err != nil {
				s.HttpLog.Errorf("Failed to unmarshal test output response: %v", err)
				continue
			}

			s.HttpLog.Errorf("Failed to send test output to controller, status code: %d, message: %v", response.StatusCode, message)
			continue
		}
	}

	if !sendSuccess {
		s.HttpLog.Errorf("Failed to send test output to controller after %d attempts.", constant.HTTP_RETRY_TIMES)
		s.HttpLog.Debugf("Test output data that failed to send: %+v", testOutput)
	}
}

func (s *httpSenderServer) setStatusIdle() {
	s.status = constant.STATUS_IDLE
}

func (s *httpSenderServer) setStatusRunning() {
	s.status = constant.STATUS_RUNNING
}

# Ezec, a exec.Cmd wrapper with syntactic sugar
It's purpose is to provide an easy way to attach consumers to process command output. An example of
this would be running ffmpeg to receive a long running RTP stream and processing the log and progress output to convert
it to metrics or provide error reporting. 


## TODO
* Types:
  * [X] CMD
  * [X] Config
  * [X] Parsers
* Methods:
  * [ ] New
  * [ ] Start/Run
  * 
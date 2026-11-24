"""stdio python 沟通示例

自定义python节点的固定协议（每行一个 JSON 对象，扁平结构）：
    Noddock → stdin
      get           {"type":"get","id":...,"value":...,"kind":...}  GetValue
      stream_in     {"type":"stream_in","id":...,"status":"ok|done","delta":"..."} (输入流块 status="done" 表示该流结束）
      run           {"type":"run","code":"<main.py 内容>"}
    stdout → Noddock
      set           {"type":"set","id":...,"value":...}  code 调 SetValue
      stream_out    {"type":"stream_out","id":...,"delta":"..."}    code 调 Stream.send
      error         {"type":"error","error":"...","traceback":"..."}  code 抛异常,error为""为正常

code（main.py）执行时 namespace 里可用：
    share_state  dict   跨请求持久状态（dict 对象常驻，修改内容跨 run 保留）
    SetValue func   设置输出参数：SetValue(param_id, value)，调用即生效，
                    一次 run 可多次调用，同一参数按序覆盖，最后一个值生效。
    GetValue func   读取输入参数：GetValue(param_id)，id 不存在返回 None。
                    stream 参数返回 Stream 对象，先取再迭代/发送：
                        s = GetValue("stream1")
                        for v in s: ...          # 输入流
                        s.send(delta)            # 输出流
"""
import sys
import json
import traceback
from threading import Thread
from queue import Queue

share_state = {}
params = {}  # get 行暂存，run 时注入，run 完成后清除

def main():
    while True:
        raw = sys.stdin.readline()
        if not raw:
            break
        raw = raw.strip()
        if not raw:
            continue
        line = json.loads(raw)
        t = line["type"]
        if t == "get":
            if line["kind"] == "stream":
                params[line["id"]] = Stream(line["id"], Queue(), emit)
            else:
                params[line["id"]] = line["value"]
        elif t == "stream_in":
            s = params.get(line["id"])
            if s is not None:
                s.queue.put(line)
        elif t == "run":
            Thread(target=execute, args=(line["code"],)).start()

def emit(msg):
    print(json.dumps(msg))
    sys.stdout.flush()

def SetValue(param_id, value):
    emit({"type": "set", "id": param_id, "value": value})

def GetValue(param_id):
    return params.get(param_id)  # id 不存在返回 None（不抛 KeyError）

def execute(code):
    try:
        ev = {"share_state": share_state,"SetValue": SetValue,"GetValue": GetValue}
        exec(code, ev)
        err, tb = "", ""
    except Exception as e:  # 只捕 RuntimeError 时，其他异常（NameError 等）会让 Go 侧永久等待
        err = str(e)
        tb = traceback.format_exc()
    # 先清除暂存 param 再发结束行：Go 收到结束行才会发下一轮的 get, 顺序保证清除不会误删下一轮暂存。
    params.clear()
    emit({"type": "error", "error": err, "traceback": tb})

class Stream:
    def __init__(self, stream_id, queue, emit):
        self.id = stream_id
        self.queue = queue
        self.emit = emit
        self._buffer = []
    # ========= Input =========
    def __iter__(self):
        return self
    def __next__(self):
        item = self.queue.get()
        if item["status"] == "done":
            raise StopIteration
        delta = item["delta"]
        self._buffer.append(delta)
        return delta
    # ========= Output =========
    def send(self, delta):
        self.emit({"type": "stream_out", "id": self.id, "delta": delta})
        self._buffer.append(delta)
    # ========= Helpers =========
    def full(self):
        return "".join(self._buffer)

if __name__ == "__main__": main()
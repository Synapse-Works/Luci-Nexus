module("luci.controller.lucinexus", package.seeall)

function index()
	if not nixio.fs.access("/etc/config/lucinexus") then
		return
	end

	entry({"admin", "services", "lucinexus"}, template("lucinexus/view"), _("LuciNexus Assistant"), 60)
	entry({"admin", "services", "lucinexus", "api"}, call("action_proxy")).leaf = true
	entry({"admin", "services", "lucinexus", "import_curl"}, call("action_import_curl")).leaf = true
end

function action_proxy()
	local http = require "luci.http"
	local content = http.content()
	local socket = require "nixio.socket"
	local client = socket.connect("127.0.0.1", 9999)
	if not client then
		http.status(503, "Service Unavailable")
		http.prepare_content("application/json")
		http.write('{"error":"LuciNexus daemon is not running."}')
		return
	end

	local raw_req = "POST /api/v1/execute HTTP/1.0\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: " .. string.len(content) .. "\r\n\r\n" .. content
	client:send(raw_req)

	local resp = ""
	repeat
		local chunk = client:recv(1024)
		if chunk and #chunk > 0 then
			resp = resp .. chunk
		end
	until not chunk or #chunk == 0
	client:close()

	local header_end = resp:find("\r\n\r\n")
	local body = header_end and resp:sub(header_end + 4) or resp
	http.prepare_content("application/json")
	http.write(body)
end

function action_import_curl()
	local http = require "luci.http"
	local sys = require "luci.sys"
	local json = require "luci.jsonc"

	local content = http.content()
	local req = json.parse(content)
	if not req or not req.curl_cmd then
		http.status(400, "Bad Request")
		http.write('{"error":"Missing curl_cmd parameter"}')
		return
	end

	local cmd = string.format('/usr/bin/lucinexus --curl %s 2>&1', sys.shellquote(req.curl_cmd))
	local out = sys.exec(cmd)
	http.prepare_content("application/json")
	http.write(json.stringify({output = out}))
end

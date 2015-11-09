#!/usr/bin/env ruby

$stdout.sync = true

p [:running, $0, *ARGV]

require 'socket'
require 'set'

require_relative './sd_daemon'

Thread.abort_on_exception = true

if !SdDaemon.listen_fds.empty?
  server = UDPSocket.for_fd(SdDaemon.listen_fds.first.fileno)
else
  server = UDPSocket.new
  server.bind 'localhost', 8000
end

closing = false

%w{INT TERM}.each do |sig|
  Signal.trap(sig) do
    server.close
    closing = true
  end
end


SdDaemon.notify_ready


while !closing
  begin
    pack, addr = server.recvfrom(10)
    p [:received, pack, addr]
  rescue Errno::EBADF # getting that when the server is closed
  end
end

puts "Stopping"

closing = false

while !closing && Thread.list.size > 1
  sleep 1
end

puts "DONE"

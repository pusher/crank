#!/usr/bin/env ruby

$stdout.sync = true

p [:running, $0, *ARGV]

require 'socket'
require 'set'

require_relative './sd_daemon'

Thread.abort_on_exception = true

if !SdDaemon.listen_fds.empty?
  server = TCPServer.for_fd(SdDaemon.listen_fds.first.fileno)
else
  server = TCPServer.open 8000
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
    Thread.new(server.accept) do |client|
      p [:conn_new, client]
      client.read
      p [:conn_old, client]
    end
  rescue Errno::EBADF # getting that when the server is closed
  end
end

puts "Stopping"

closing = false

while !closing && Thread.list.size > 1
  sleep 1
end

puts "DONE"

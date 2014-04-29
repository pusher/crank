#!/usr/bin/env ruby

$stdout.sync = true

p [:running, $0, *ARGV]

require 'socket'
require 'set'

require 'bundler/setup'
require 'eventmachine'

require_relative './sd_daemon'

class ServerSupervisor
  # Delegate must respond to
  # * start_accepting(fd)
  # * start_server(port)
  # * close_gracefully(&onempty)
  # * close_forcefully(&onempty)
  def initialize(server_delegate, port)
    @server = server_delegate
    @port = port
    @accepting = false
    @stop_gracefully = true

    @fds = SdDaemon.listen_fds
  end

  def run
    start_accepting()

    puts "READY"
    SdDaemon.notify_ready
  end

  def start_accepting
    return if @accepting

    if @fds.any?
      @server.start_fd(@fds.first)
    else
      @server.start_port(@port)
    end
    @accepting = true
  end

  # Simplified wrapper for close_gracefully & close_forcefully
  # Calls
  def stop(&onempty)
    if @stop_gracefully
      puts "RUBY: INT/TERM: Closing connections gracefully (#{@server.report})"
      @server.close_gracefully(&onempty)
      @stop_gracefully = false
    else
      puts "RUBY: INT/TERM: Closing connections forcefully (#{@server.report})"
      @server.close_forcefully(&onempty)
    end
  end

  def stop_accepting(&onempty)
    return unless @accepting

    @server.stop_accepting(&onempty)
    @accepting = false
    register_onempty(onempty) if onempty
  end
end


class Server
  def initialize(handler_klass, handler_options)
    @handler_klass, @handler_options = handler_klass, handler_options
    @connections = Set.new
    @onempty_callback = nil

    handler_options[:server] = self
  end

  def conn_add(c)
    @connections.add(c)
    puts report
  end

  def conn_rem(c)
    @connections.delete(c)
    puts report
    @onempty_callback.call if @onempty_callback && @connections.blank?
  end

  def report
    "Connections open: #{@connections.size}"
  end

  def start_fd(fd)
    puts "RUBY: Binding app to passed file descriptor"
    @server = EM.attach_server(fd, @handler_klass, @handler_options)
  end

  def start_port(port)
    puts "RUBY: Starting new server on port #{port}"
    @server = EM.start_server('0.0.0.0', port, @handler_klass, @handler_options)
  end

  def stop_accepting(&onempty)
    return unless @server

    EM.stop_server(@server)
    @server = nil
    register_onempty(onempty) if onempty
  end

  def close_gracefully(&onempty)
    @connections.each { |c| c.close_gracefully }
    register_onempty(onempty) if onempty
  end

  def close_forcefully(&onempty)
    @connections.each { |c| c.close_forcefully }
    register_onempty(onempty) if onempty
  end

  private

  def register_onempty(blk)
    @connections.empty? ? blk.call : @onempty_callback = blk
  end
end


class AppHandler < EM::Connection
  def initialize(options)
    @server = options[:server]
  end

  def post_init
    @server.conn_add(self)
    send_data("Hello there (#{Process.pid})")
    # @timer = EM::Timer.new(0.2) do
    #   close_connection
    # end
  end

  def unbind
    # @timer.cancel if @timer
    @server.conn_rem(self)
  end

  def close_gracefully
    send_data("close handshake")
  end

  def close_forcefully
    close_connection
  end
end

server = Server.new(AppHandler, {})
supervisor = ServerSupervisor.new(server, 8000)

EM.run do
  supervisor.run

  %w{INT TERM}.each do |sig|
    Signal.trap(sig) do
      supervisor.stop { puts "RUBY: Graceful exit"; EM.stop }
    end
  end
end

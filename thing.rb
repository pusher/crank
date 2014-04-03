require 'bundler/setup'
require 'eventmachine'
require 'set'
require 'socket'

# Implements sd-notify
# http://www.freedesktop.org/software/systemd/man/sd_notify.html
module Sd extend self
  class NullSocket
    def noop(*) end
    alias sendmsg noop
    alias close_on_exec= noop
  end

  # MSG_NOSIGNAL doesn't exist on OSX
  # It's used to avoid SIGPIPE on the process if the other end disappears
  MSG_NOSIGNAL = Socket.const_defined?(:MSG_NOSIGNAL) ? Socket::MSG_NOSIGNAL : 0

  def notify(msg)
    socket.sendmsg cleanup(msg), MSG_NOSIGNAL
  end

  def notify_ready
    notify "READY=1"
  end

  def notify_status(msg)
    notify "STATUS=#{msg}"
  end

  def notify_errno(errno)
    notify "ERRNO=#{errno}"
  end

  def notify_buserror(err)
    notify "BUSERROR=#{err}"
  end

  def notify_mainpid(pid = Process.pid)
    notify "MAINPID=#{pid}"
  end

  def notify_watchdog
    notify "WATCHDOG=1"
  end

  def watchdog_enabled?
    memoize(:watchdog_enabled?) do
      break if ENV.has_key?('WATCHDOG_PID') && ENV.has_key?('WATCHDOG_USEC')
      break if ENV['WATCHDOG_PID'].to_i != Process.pid
      break if ENV['WATCHDOG_USEC'].to_i <= 0
      ENV.delete 'WATCHDOG_PID'
      true
    end
  end

  def watchdog_usec
    memoize(:watchdog_usec, watchdog_enabled? && ENV.delete('WATCHDOG_USEC').to_i)
  end

  protected

  def socket
    socket = if ((socket_path = ENV.delete('NOTIFY_SOCKET')))
      UNIXSocket.open(socket_path)
    # This is our own extension
    elsif ((fd = ENV['NOTIFY_FD'])) # && ENV['NOTIFY_PID'].to_i == Process.pid)
      UNIXSocket.for_fd fd.to_i
    else
      NullSocket.new
    end
    socket.close_on_exec = true
    memoize(:socket, socket)
  end

  def cleanup(msg)
    # Ensure msg doesn't contain a \n
    msg.gsub("\n", '')
  end

  def memoize(method, value=nil)
    value = yield if block_given?
    singleton_class.send(:define_method, method) { value }
    value
  end
end

class CrankedServer
  FD = 3

  attr_reader :under_crank

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

    @under_crank = ENV["LISTEN_FDS"] && ENV["LISTEN_FDS"].to_i == 1
  end

  def run
    # Fallback to starting accepting immediately in the absence of crank
    start_accepting()

    Sd.notify_ready
  end

  def start_accepting
    return if @accepting

    if @under_crank
      @server.start_accepting(FD)
    else
      @server.start_server(@port)
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
  end

  def conn_rem(c)
    @connections.delete(c)
    @onempty_callback.call if @onempty_callback
  end

  def report
    "Connections open: #{@connections.size}"
  end

  def start_accepting(fd)
    puts "RUBY: Binding app to passed file descriptor"
    @server = EM.attach_server(fd, @handler_klass, @handler_options)
  end

  def start_server(port)
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
cranked_server = CrankedServer.new(server, 8000)

EM.run do
  cranked_server.run

  %w{INT TERM}.each do |sig|
    Signal.trap(sig) do
      cranked_server.stop { puts "RUBY: Graceful exit"; EM.stop }
    end
  end
end

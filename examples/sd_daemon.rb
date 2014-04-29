# A pure ruby implementation of the sd-daemon library distributed with SystemD.
#
# Includes:
#  * File descriptor passing for socket-based activation
#  * Daemon startup and status notification
#  * Watchdog system
#  *
#
# Missing:
#  * Support for logging with log levels on stderr
#  * Detection of systemd boots
#  * Detection of FD types
# 
# More details: http://www.freedesktop.org/software/systemd/man/sd-daemon.html
#
module SdDaemon extend self
  class NullSocket
    def noop(*) end
    alias sendmsg noop
    alias close_on_exec= noop
  end

  # MSG_NOSIGNAL doesn't exist on OSX
  # It's used to avoid SIGPIPE on the process if the other end disappears
  MSG_NOSIGNAL = Socket.const_defined?(:MSG_NOSIGNAL) ? Socket::MSG_NOSIGNAL : 0

  LISTEN_FDS_START = 3

  # Sends a message to the supervisor if LISTEN_FD/LISTEN_SOCKET is set.
  # Otherwise it is a noop.
  #
  # LISTEN_FD is not part of the original spec and an extension for crank.
  def notify(msg)
    notify_socket.sendmsg cleanup(msg), MSG_NOSIGNAL
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

  # Returns true if the supervisor requests a watchdog
  def watchdog_enabled?
    memoize(:watchdog_enabled?) do
      break if ENV.has_key?('WATCHDOG_PID') && ENV.has_key?('WATCHDOG_USEC')
      break if ENV['WATCHDOG_PID'].to_i != Process.pid
      break if ENV['WATCHDOG_USEC'].to_i <= 0
      ENV.delete 'WATCHDOG_PID'
      true
    end
  end

  # Returns how often the supervisor expects watchdog notifications
  def watchdog_usec
    memoize(:watchdog_usec, watchdog_enabled? && ENV.delete('WATCHDOG_USEC').to_i)
  end

  # Returns an array of IO if LISTEN_FDS is set.
  #
  # The crank_compat flag turns off the LISTEN_PID check when true.
  def listen_fds(crank_compat = true)
    fds = []
    if (crank_compat || ENV['LISTEN_PID'].to_i == Process.pid) &&
       (fd_count = ENV['LISTEN_FDS'].to_i) > 0
      ENV.delete('LISTEN_PID')
      ENV.delete('LISTEN_FDS')
      fds = fd_count.times
        .map{|i| IO.new(LISTEN_FDS_START + i)}
        .each{|io| io.close_on_exec = true }
    end
    memoize(:listen_fds, fds)
  end

  protected

  def notify_socket
    socket = if ((socket_path = ENV.delete('NOTIFY_SOCKET')))
      UNIXSocket.open(socket_path)
    # This is our own extension
    elsif ((fd = ENV['NOTIFY_FD'])) # && ENV['NOTIFY_PID'].to_i == Process.pid)
      UNIXSocket.for_fd fd.to_i
    else
      NullSocket.new
    end
    socket.close_on_exec = true
    memoize(:notify_socket, socket)
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

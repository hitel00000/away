use strict;
use warnings;

use Irssi;
use IO::Socket::UNIX;
use Time::HiRes qw(time);
use Fcntl qw(O_RDONLY O_NONBLOCK);

our $VERSION = '0.2';

our %IRSSI = (
  authors     => 'Away',
  name        => 'away_bridge',
  description => 'C-001/C-002 irssi bridge',
  license     => 'MIT',
);

#
# -------------------------
# paths
# -------------------------
#

my $EVENT_SOCKET = "/tmp/away/irc-companion.sock";
my $CMD_FIFO     = "/tmp/away/irc-companion.cmd";

my $cmd_fh;

#
# -------------------------
# utility
# -------------------------
#

sub event_id {
  return "evt_" . int(time()*1000) . "_" . int(rand(100000));
}

sub iso_ts {
  my @t = gmtime();
  return sprintf(
      "%04d-%02d-%02dT%02d:%02d:%02dZ",
      $t[5]+1900,$t[4]+1,$t[3],
      $t[2],$t[1],$t[0]
  );
}

sub json_escape {
  my ($s)=@_;
  $s='' unless defined $s;

  $s =~ s/\\/\\\\/g;
  $s =~ s/"/\\"/g;
  $s =~ s/\n/\\n/g;
  $s =~ s/\r/\\r/g;
  $s =~ s/\t/\\t/g;

  return $s;
}

#
# -------------------------
# outbound transport
# -------------------------
#

sub emit_json {
  my ($line)=@_;

  my $sock = IO::Socket::UNIX->new(
      Type => SOCK_STREAM,
      Peer => $EVENT_SOCKET
  );

  unless ($sock) {
      Irssi::print("away_bridge: relay socket unavailable");
      return;
  }

  print $sock $line . "\n";
  close($sock);
}

#
# -------------------------
# event builders
# -------------------------
#

sub public_event_json {

  my ($server,$target,$nick,$text)=@_;

  return sprintf(
'{"type":"message.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","buffer_id":"chan:%s","buffer_type":"channel","nick":"%s","text":"%s","highlight":false,"tags":[]}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($target),
      json_escape($nick),
      json_escape($text),
  );
}

sub private_event_json {

  my ($server,$nick,$text)=@_;

  return sprintf(
'{"type":"dm.created","version":1,"id":"%s","timestamp":"%s","payload":{"network":"%s","peer":"%s","text":"%s"}}',
      event_id(),
      iso_ts(),
      json_escape($server->{tag}),
      json_escape($nick),
      json_escape($text),
  );
}

#
# -------------------------
# irssi signal hooks
# -------------------------
#

sub on_public_message {

  my ($server,$text,$nick,$address,$target)=@_;

  emit_json(
      public_event_json(
          $server,
          $target,
          $nick,
          $text
      )
  );
}

sub on_private_message {

  my ($server,$text,$nick,$address)=@_;

  emit_json(
      private_event_json(
          $server,
          $nick,
          $text
      )
  );
}

#
# -------------------------
# inbound command fifo
# -------------------------
#

sub init_command_fifo {

  unless (-p $CMD_FIFO) {
      system("mkfifo",$CMD_FIFO);
  }

  sysopen(
      $cmd_fh,
      $CMD_FIFO,
      O_RDONLY | O_NONBLOCK
  ) or do {
      Irssi::print("away_bridge: fifo open failed");
      return;
  };

  Irssi::print("away_bridge command fifo ready");
}

sub parse_send_message {

  my ($line)=@_;

  return unless $line =~ /"action":"send_message"/;

  return unless $line =~ /"target":"([^"]+)"/;
  my $target=$1;

  return unless $line =~ /"text":"([^"]*)"/;
  my $text=$1;

  $text =~ s/\\"/"/g;
  $text =~ s/\\\\/\\/g;

  return ($target,$text);
}

sub poll_commands {

  return 1 unless $cmd_fh;

  my $buf='';

  my $n = sysread(
      $cmd_fh,
      $buf,
      4096
  );

  return 1 unless defined $n;
  return 1 if $n <= 0;

  for my $line (split /\n/, $buf) {
  
      my ($target,$text)=parse_send_message($line);
      next unless $target;
  
      my @servers = Irssi::servers();
      next unless @servers;
  
      $servers[0]->command(
         "msg $target $text"
      );
  }

  return 1;
}

#
# -------------------------
# init
# -------------------------
#

Irssi::signal_remove(
  'message public',
  'on_public_message'
);

Irssi::signal_remove(
  'message private',
  'on_private_message'
);

Irssi::signal_add(
  'message public',
  'on_public_message'
);

Irssi::signal_add(
  'message private',
  'on_private_message'
);

init_command_fifo();

Irssi::timeout_add(
  250,
  'poll_commands',
  undef
);

Irssi::print("away_bridge loaded");
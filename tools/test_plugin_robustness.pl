#!/usr/bin/env perl
use strict;
use warnings;
use IO::Socket::UNIX;
use File::Temp qw(tempdir);
use File::Spec;
use Test::More tests => 5;

# Mock Irssi
package Irssi {
    sub print {
        my ($msg) = @_;
    }
    sub timeout_add {}
    sub signal_add {}
    sub signal_remove {}
}

my $tmpdir = tempdir(CLEANUP => 1);
my $sock_path = File::Spec->catfile($tmpdir, "test.sock");

# Load the plugin logic
my $plugin_file = 'irssi-plugin/away_bridge.pl';
my $plugin_code = do {
    local $/;
    open my $fh, '<', $plugin_file or die "Could not open $plugin_file: $!";
    <$fh>;
};

# Patch the socket path in the code for testing
$plugin_code =~ s/my \$EVENT_SOCKET\s*=\s*".*";/my \$EVENT_SOCKET = "$sock_path";/;
# Remove Irssi calls at the bottom that would start timers/hooks
$plugin_code =~ s/init_command_fifo.*//s;

# Wrap in a package to avoid collisions if necessary, but here we just eval
eval $plugin_code;
if ($@) {
    die "Eval failed: $@";
}

# TEST 1: Send message while relay is down
main::emit_json('{"test":"msg1"}');
is(scalar(@main::outbound_queue), 1, "Message 1 queued when relay is down");

# TEST 2: Send another message
main::emit_json('{"test":"msg2"}');
is(scalar(@main::outbound_queue), 2, "Message 2 queued when relay is down");

# TEST 3: Start relay and flush
my $server = IO::Socket::UNIX->new(
    Local => $sock_path,
    Listen => 5,
    Type => SOCK_STREAM,
) or die "Could not create server socket at $sock_path: $!";

main::flush_queue();
is(scalar(@main::outbound_queue), 0, "Queue flushed when relay is up");

# Verify received data
my $client = $server->accept();
my $data1 = <$client>;
like($data1, qr/msg1/, "Relay received message 1");
my $data2 = <$client>;
like($data2, qr/msg2/, "Relay received message 2");

$client->close();
$server->close();
unlink($sock_path);

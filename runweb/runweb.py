#!/usr/bin/env python3
"""runweb — serve an HTML file or a folder on your local network.

Usage:
  runweb page.html          serve page.html (and its folder)
  runweb ./site             serve a folder (index.html if present)
  runweb                    serve the current folder
Options:
  -p, --port N   starting port (default 8000, auto-bumps if busy)
  -o, --open     also open it in your browser
  -q, --qr       print a QR code for phones (needs `qrencode`)
  -s, --https    serve over HTTPS with a self-signed cert
"""
import argparse
import functools
import http.server
import os
import shutil
import socket
import socketserver
import ssl
import subprocess
import sys
import webbrowser
from urllib.parse import quote


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        # No caching, so edits show up on refresh
        self.send_header("Cache-Control", "no-store")
        super().end_headers()

    def log_message(self, fmt, *args):
        sys.stderr.write(f"  {self.address_string():<15} {fmt % args}\n")


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True
    allow_reuse_address = True

    def handle_error(self, request, client_address):
        # Browsers drop the connection when they first see the self-signed
        # cert; don't spam tracebacks for that.
        err = sys.exc_info()[1]
        if isinstance(err, (ssl.SSLError, ConnectionError)):
            return
        super().handle_error(request, client_address)


CERT_DIR = os.path.expanduser("~/.config/runweb")


def ensure_cert(ips):
    """Make (or reuse) a self-signed cert covering localhost + current IPs."""
    os.makedirs(CERT_DIR, exist_ok=True)
    cert = os.path.join(CERT_DIR, "cert.pem")
    key = os.path.join(CERT_DIR, "key.pem")
    stamp = os.path.join(CERT_DIR, "ips.txt")
    want = ",".join(sorted(ips))

    if all(os.path.exists(p) for p in (cert, key, stamp)):
        with open(stamp) as f:
            if f.read() == want:
                return cert, key

    if not shutil.which("openssl"):
        sys.exit("runweb: --https needs openssl installed")

    san = "DNS:localhost,IP:127.0.0.1" + "".join(f",IP:{ip}" for ip in ips)
    print("  generating HTTPS certificate...")
    subprocess.run(
        ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
         "-keyout", key, "-out", cert, "-days", "825",
         "-subj", "/CN=runweb", "-addext", f"subjectAltName={san}"],
        check=True, capture_output=True)
    os.chmod(key, 0o600)
    with open(stamp, "w") as f:
        f.write(want)
    return cert, key


def local_ips():
    ips = []
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("10.255.255.255", 1))
        ips.append(s.getsockname()[0])
        s.close()
    except OSError:
        pass
    try:
        out = subprocess.run(["hostname", "-I"], capture_output=True,
                             text=True, timeout=2).stdout.split()
        for ip in out:
            if ":" not in ip and not ip.startswith("127.") and ip not in ips:
                ips.append(ip)
    except Exception:
        pass
    return ips


def is_tailscale(ip):
    a, b = (int(x) for x in ip.split(".")[:2])
    return a == 100 and 64 <= b <= 127


def bind(root, start_port):
    handler = functools.partial(Handler, directory=root)
    for port in range(start_port, start_port + 50):
        try:
            return Server(("0.0.0.0", port), handler), port
        except OSError:
            continue
    sys.exit(f"runweb: no free port between {start_port} and {start_port + 49}")


def main():
    ap = argparse.ArgumentParser(prog="runweb", add_help=True,
                                 description="Serve an HTML file or folder on your LAN.")
    ap.add_argument("target", nargs="?", default=".")
    ap.add_argument("-p", "--port", type=int, default=8000)
    ap.add_argument("-o", "--open", action="store_true")
    ap.add_argument("-q", "--qr", action="store_true")
    ap.add_argument("-s", "--https", action="store_true")
    args = ap.parse_args()

    target = os.path.abspath(os.path.expanduser(args.target))
    if not os.path.exists(target):
        sys.exit(f"runweb: '{args.target}' doesn't exist")

    if os.path.isfile(target):
        root, entry = os.path.dirname(target), "/" + quote(os.path.basename(target))
    else:
        root, entry = target, "/"

    ips = local_ips()
    httpd, port = bind(root, args.port)
    scheme = "http"

    if args.https:
        cert, key = ensure_cert(ips)
        ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        ctx.load_cert_chain(cert, key)
        httpd.socket = ctx.wrap_socket(httpd.socket, server_side=True,
                                       do_handshake_on_connect=False)
        scheme = "https"

    print(f"\n  runweb · serving {root}\n")
    print(f"  This PC:    {scheme}://localhost:{port}{entry}")
    lan_url = None
    for ip in ips:
        label = "Tailscale:" if is_tailscale(ip) else "Network:  "
        url = f"{scheme}://{ip}:{port}{entry}"
        print(f"  {label}  {url}")
        if lan_url is None and not is_tailscale(ip):
            lan_url = url
    print("\n  Ctrl+C to stop\n")

    if args.qr and lan_url:
        if shutil.which("qrencode"):
            subprocess.run(["qrencode", "-t", "ANSIUTF8", lan_url])
        else:
            print("  (install qrencode for QR codes)\n")

    if args.open:
        webbrowser.open(f"{scheme}://localhost:{port}{entry}")

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n  stopped.")
    finally:
        httpd.server_close()


if __name__ == "__main__":
    main()

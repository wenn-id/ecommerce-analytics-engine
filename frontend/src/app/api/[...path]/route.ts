import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';
const SERVER_API_KEY = process.env.API_KEY || '';

export async function GET(request: NextRequest, { params }: { params: { path: string[] } }) {
  return proxyRequest(request, params.path);
}

export async function POST(request: NextRequest, { params }: { params: { path: string[] } }) {
  return proxyRequest(request, params.path);
}

export async function PUT(request: NextRequest, { params }: { params: { path: string[] } }) {
  return proxyRequest(request, params.path);
}

export async function DELETE(request: NextRequest, { params }: { params: { path: string[] } }) {
  return proxyRequest(request, params.path);
}

// Resolve the client IP from standard proxy headers. `request.ip` is
// deprecated since Next.js 14 and may be undefined, which used to collapse
// every proxied request onto 127.0.0.1 (#63).
function resolveClientIp(request: NextRequest): string {
  const forwardedFor = request.headers.get('x-forwarded-for');
  if (forwardedFor) {
    const firstHop = forwardedFor.split(',')[0].trim();
    if (firstHop) return firstHop;
  }
  const realIp = request.headers.get('x-real-ip');
  if (realIp) return realIp.trim();
  return '127.0.0.1';
}

async function proxyRequest(request: NextRequest, pathSegments: string[]) {
  const path = (pathSegments || []).join('/');
  const searchParams = request.nextUrl.search;
  const targetUrl = `${BACKEND_URL}/${path}${searchParams}`;

  const headers = new Headers();
  headers.set('Content-Type', 'application/json');
  // Only the server-side key is ever forwarded. When it is unset the request
  // is sent without a key: the backend is then intentionally keyless (dev
  // default), and a browser-supplied key is never trusted (#63).
  if (SERVER_API_KEY) {
    headers.set('X-API-Key', SERVER_API_KEY);
  }

  const requestedWith = request.headers.get('x-requested-with') || 'XMLHttpRequest';
  headers.set('X-Requested-With', requestedWith);

  const csrfToken = request.headers.get('x-csrf-token');
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  headers.set('X-Forwarded-For', resolveClientIp(request));

  const reqInit: RequestInit = {
    method: request.method,
    headers,
  };

  if (request.method !== 'GET' && request.method !== 'HEAD') {
    const body = await request.text();
    if (body) {
      reqInit.body = body;
    }
  }

  try {
    const response = await fetch(targetUrl, reqInit);
    const data = await response.text();

    const responseHeaders = new Headers();
    responseHeaders.set('Content-Type', response.headers.get('Content-Type') || 'application/json');

    const cacheControl = response.headers.get('Cache-Control');
    if (cacheControl) {
      responseHeaders.set('Cache-Control', cacheControl);
    }
    const etag = response.headers.get('ETag');
    if (etag) {
      responseHeaders.set('ETag', etag);
    }
    const vary = response.headers.get('Vary');
    if (vary) {
      responseHeaders.set('Vary', vary);
    }

    return new NextResponse(data, {
      status: response.status,
      headers: responseHeaders,
    });
  } catch (error) {
    return NextResponse.json(
      { error: { message: 'Failed to proxy request to backend analytics engine' } },
      { status: 502 }
    );
  }
}

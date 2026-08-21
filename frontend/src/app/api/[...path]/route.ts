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

async function proxyRequest(request: NextRequest, pathSegments: string[]) {
  const path = (pathSegments || []).join('/');
  const searchParams = request.nextUrl.search;
  const targetUrl = `${BACKEND_URL}/${path}${searchParams}`;

  const headers = new Headers();
  headers.set('Content-Type', 'application/json');
  if (SERVER_API_KEY) {
    headers.set('X-API-Key', SERVER_API_KEY);
  }

  const clientIp = request.ip || '127.0.0.1';
  headers.set('X-Forwarded-For', clientIp);

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
    return new NextResponse(data, {
      status: response.status,
      headers: {
        'Content-Type': response.headers.get('Content-Type') || 'application/json',
      },
    });
  } catch (error) {
    return NextResponse.json(
      { error: { message: 'Failed to proxy request to backend analytics engine' } },
      { status: 502 }
    );
  }
}

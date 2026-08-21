import React from 'react';
import './globals.css';

export const metadata = {
  title: 'E-Commerce Analytics Engine',
  description: 'Multi-channel marketing & e-commerce analytics dashboard',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}

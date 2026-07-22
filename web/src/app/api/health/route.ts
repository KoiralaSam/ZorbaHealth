import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

/** Lightweight probe target — homepage SSR is too slow under cluster CPU limits. */
export async function GET() {
  return NextResponse.json({ status: "ok" }, { status: 200 });
}

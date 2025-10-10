const latencyHosts = [
  'netlib-ping-africa.poki.io',
  'netlib-ping-asia-northeast.poki.io',
  'netlib-ping-asia-south.poki.io',
  'netlib-ping-asia-southeast.poki.io',
  'netlib-ping-australia.poki.io',
  'netlib-ping-eu-north.poki.io',
  'netlib-ping-eu-west.poki.io',
  'netlib-ping-me-central.poki.io',
  'netlib-ping-south-america.poki.io',
  'netlib-ping-us-central.poki.io',
  'netlib-ping-us-east.poki.io'
]

export async function getLatencyVector (max: number, pings: number): Promise<number[]> {
  const measurements = await Promise.all(latencyHosts.map(async (host) => {
    let latency = 0

    for (let i = 0; i < pings; i++) {
      const start = performance.now()
      try {
        await fetch(`https://${host}/`, {
          method: 'HEAD',
          cache: 'no-store',
          mode: 'no-cors',
          signal: AbortSignal.timeout(max)
        })
      } catch {}

      latency += Math.round((performance.now() - start) / 2) // Divide by 2 to estimate one-way latency.
    }

    return Math.round(latency / pings) // Average of two measurements.
  }))
  return measurements
}

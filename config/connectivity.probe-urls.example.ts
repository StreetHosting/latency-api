/**
 * Reference for updating the Next.js site after probe deploy.
 * Copy values into config/connectivity.ts in the streethosting site repo.
 */
export const probeTargetsExample = [
  {
    networkId: "sp-games",
    targets: [
      {
        probeUrl: "https://latency-sp-games-1.streethosting.com.br/ping",
        displayAddress: "177.55.0.10",
      },
      {
        probeUrl: "https://latency-sp-games-2.streethosting.com.br/ping",
        displayAddress: "177.55.0.11",
      },
    ],
  },
  {
    networkId: "sp-empresa",
    targets: [
      {
        probeUrl: "https://latency-sp-empresa-1.streethosting.com.br/ping",
        displayAddress: "177.56.0.10",
      },
      {
        probeUrl: "https://latency-sp-empresa-2.streethosting.com.br/ping",
        displayAddress: "177.56.0.11",
      },
    ],
  },
  {
    networkId: "sp-nao-mitigada",
    targets: [
      {
        probeUrl: "https://latency-sp-raw-1.streethosting.com.br/ping",
        displayAddress: "177.57.0.10",
      },
      {
        probeUrl: "https://latency-sp-raw-2.streethosting.com.br/ping",
        displayAddress: "177.57.0.11",
      },
    ],
  },
] as const;

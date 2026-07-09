// SPIKE 15-01 — THROWAWAY, delete after the file:// glyph gate resolves.
//
// Phase 15 risk-gate spike (D-01/D-02/D-03, RESEARCH.md "Spike Design"):
// prove — or disprove — that MapLibre GL Native resolves a `file://`-scheme
// glyph URL TEMPLATE and a `file://`-scheme sprite base at runtime, rendering a
// place-name label (A1) and a sprite icon (A2) from files pre-seeded into the
// app documents directory, on a PHYSICAL device in airplane mode.
//
// This screen builds a MINIMAL hand-written style JSON (NOT the 7,677-line
// theme): one GeoJSON point with a `name`, a `file://` `glyphs` template, a
// `file://` `sprite`, and two symbol layers (label + `arrow` icon). No basemap
// tile source is needed — a labelled point over a flat background is enough to
// prove glyph/sprite resolution.
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/spike_glyph_seed.dart';

// A fixed point to hang the label + icon on (central Alps — arbitrary).
const _spikeLat = 47.0;
const _spikeLon = 11.0;

class SpikeGlyphFileScreen extends ConsumerStatefulWidget {
  const SpikeGlyphFileScreen({super.key});

  @override
  ConsumerState<SpikeGlyphFileScreen> createState() =>
      _SpikeGlyphFileScreenState();
}

class _SpikeGlyphFileScreenState extends ConsumerState<SpikeGlyphFileScreen> {
  String? _styleJson;
  List<String> _log = const [];
  Object? _error;

  @override
  void initState() {
    super.initState();
    _seedThenBuildStyle();
  }

  Future<void> _seedThenBuildStyle() async {
    try {
      final result = await seedSpikeGlyphCache(ref);
      if (!mounted) return;
      setState(() {
        _styleJson = _buildSpikeStyle(result.cacheDir);
        _log = result.log;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  /// Hand-build the minimal style JSON with `file://` glyph + sprite URLs rooted
  /// at the resolved app-docs cache dir. The `{fontstack}`/`{range}` tokens are
  /// left LITERAL — MapLibre GL Native substitutes them at runtime (this is
  /// exactly the resolution behaviour the gate tests).
  String _buildSpikeStyle(String cacheDir) {
    final style = <String, dynamic>{
      'version': 8,
      'name': 'spike-15-01',
      // A1 — the risk gate: file:// glyph URL TEMPLATE.
      'glyphs': 'file://$cacheDir/{fontstack}/{range}.pbf',
      // A2 — same scheme for the sprite base.
      'sprite': 'file://$cacheDir/sprite',
      'sources': {
        'spike-point': {
          'type': 'geojson',
          'data': {
            'type': 'Feature',
            'geometry': {
              'type': 'Point',
              'coordinates': [_spikeLon, _spikeLat],
            },
            'properties': {'name': 'Wanderer Spike'},
          },
        },
      },
      'layers': [
        {
          'id': 'spike-bg',
          'type': 'background',
          'paint': {'background-color': '#e8e8e8'},
        },
        // A2 — sprite icon from file:// sprite ("arrow" is a real icon in the
        // Protomaps v4 sprite; verified present).
        {
          'id': 'spike-icon',
          'type': 'symbol',
          'source': 'spike-point',
          'layout': {
            'icon-image': 'arrow',
            'icon-size': 2.0,
            'icon-allow-overlap': true,
          },
        },
        // A1 — the place-name label from file:// glyphs.
        {
          'id': 'spike-label',
          'type': 'symbol',
          'source': 'spike-point',
          'layout': {
            'text-field': ['get', 'name'],
            'text-font': ['Noto Sans Regular'],
            'text-size': 20.0,
            'text-anchor': 'top',
            'text-offset': [0.0, 1.5],
            'text-allow-overlap': true,
          },
          'paint': {'text-color': '#111111'},
        },
      ],
    };
    return jsonEncode(style);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('SPIKE 15-01 · file:// glyph gate')),
      body: _error != null
          ? _StatusPanel(
              title: 'Seed failed (still online?)',
              lines: ['$_error'],
            )
          : _styleJson == null
          ? const Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  CircularProgressIndicator(),
                  SizedBox(height: 16),
                  Text('Seeding glyph + sprite cache (needs network)…'),
                ],
              ),
            )
          : Stack(
              children: [
                MapLibreMap(
                  options: MapOptions(
                    initStyle: _styleJson!,
                    initCenter: const Geographic(lat: _spikeLat, lon: _spikeLon),
                    initZoom: 4,
                  ),
                ),
                Positioned(
                  left: 0,
                  right: 0,
                  bottom: 0,
                  child: _StatusPanel(
                    title:
                        'Cache seeded — enable AIRPLANE MODE, then reopen this screen.',
                    lines: _log,
                  ),
                ),
              ],
            ),
    );
  }
}

class _StatusPanel extends StatelessWidget {
  const _StatusPanel({required this.title, required this.lines});

  final String title;
  final List<String> lines;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      color: Colors.black.withValues(alpha: 0.72),
      padding: const EdgeInsets.all(12),
      constraints: const BoxConstraints(maxHeight: 220),
      child: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: const TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 8),
            for (final line in lines)
              SelectableText(
                line,
                style: const TextStyle(
                  color: Colors.white70,
                  fontSize: 12,
                  fontFamily: 'monospace',
                ),
              ),
          ],
        ),
      ),
    );
  }
}

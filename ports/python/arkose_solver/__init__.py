"""Pure-Python Arkose Labs / FunCaptcha solver.

    from arkose_solver import Solver

    s = Solver(
        surl="https://verify.example.com",
        public_key="00000000-0000-0000-0000-000000000000",
        site="https://www.example.com",
    )
    res = s.solve()
    print(res.token, res.suppressed)

The site's RSA public key is extracted from the served api.js on every solve,
so there is nothing to capture or pin.
"""

from .derived import (
    audio_codecs_extended_hash,
    compute_browser_object_checks,
    compute_f58835f,
    compute_speech,
    compute_webgl_extensions_hash,
    video_codecs_extended_hash,
)
from .devices import generate_device, pool_sizes
from .hashes import compute_f, compute_ife_hash, khash
from .solver import Solver, SolveResult, solve
from .vmkey import extract_rsa_key
from .webgl import build_webgl_fields, compute_rtt_type, compute_webgl_hash

__version__ = "1.1.0"

__all__ = [
    "Solver", "SolveResult", "solve",
    "extract_rsa_key",
    "khash", "compute_f", "compute_ife_hash",
    "build_webgl_fields", "compute_webgl_hash", "compute_rtt_type",
    "compute_f58835f", "compute_browser_object_checks", "compute_speech",
    "compute_webgl_extensions_hash",
    "audio_codecs_extended_hash", "video_codecs_extended_hash",
    "generate_device", "pool_sizes",
]

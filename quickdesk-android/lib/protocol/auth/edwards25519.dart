/// edwards25519.dart - 纯 Dart 的 edwards25519 点运算
///
/// 为 SPAKE2 (BoringSSL spake25519 变体) 提供底层点运算，
/// 语义对照 @noble/curves ed25519 的 ExtendedPoint（WebClient spake2.js 所用）。
///
/// 仅实现 SPAKE2 所需的最小集合：
///   解压/压缩 (fromBytes/toBytes)、加法、取反、点倍增、标量乘。
///
/// 注意：本实现使用 BigInt，非常量时间。SPAKE2 的私钥是一次性会话随机数，
/// 若未来对侧信道有更高要求可替换为经审计的常量时间库。
/// 素数域 p = 2^255 - 19
library;

final BigInt curveP = (BigInt.one << 255) - BigInt.from(19);

/// 基点群的阶 L = 2^252 + 27742317777372353535851937790883648493
final BigInt curveOrder = (BigInt.one << 252) +
    BigInt.parse('27742317777372353535851937790883648493');

/// 扭曲 Edwards 曲线参数 d = -121665/121666 mod p
final BigInt curveD = _mod(-BigInt.from(121665) * _inv(BigInt.from(121666)));

/// sqrt(-1) mod p = 2^((p-1)/4) mod p
final BigInt sqrtM1 = BigInt.two.modPow((curveP - BigInt.one) >> 2, curveP);

BigInt _mod(BigInt a) {
  final r = a % curveP;
  return r.isNegative ? r + curveP : r;
}

BigInt _inv(BigInt a) => a.modPow(curveP - BigInt.two, curveP);

/// 扩展坐标 (X, Y, Z, T)，x = X/Z, y = Y/Z, T = XY/Z
/// 曲线方程 (a = -1): -x^2 + y^2 = 1 + d*x^2*y^2
class EdwardsPoint {
  final BigInt x, y, z, t;

  const EdwardsPoint(this.x, this.y, this.z, this.t);

  /// 单位元 (0, 1)
  static final EdwardsPoint identity =
      EdwardsPoint(BigInt.zero, BigInt.one, BigInt.one, BigInt.zero);

  /// 基点 G (RFC 8032)
  static final EdwardsPoint base = () {
    final gy = _mod(BigInt.from(4) * _inv(BigInt.from(5)));
    final gx = _recoverX(gy, 0);
    return EdwardsPoint(gx, gy, BigInt.one, _mod(gx * gy));
  }();

  /// 点加法 add-2008-hwcd-3 (a = -1)
  EdwardsPoint add(EdwardsPoint o) {
    final a = _mod((y - x) * (o.y - o.x));
    final b = _mod((y + x) * (o.y + o.x));
    final c = _mod(BigInt.two * curveD * t * o.t);
    final d = _mod(BigInt.two * z * o.z);
    final e = _mod(b - a);
    final f = _mod(d - c);
    final g = _mod(d + c);
    final h = _mod(b + a);
    return EdwardsPoint(_mod(e * f), _mod(g * h), _mod(f * g), _mod(e * h));
  }

  /// 点倍增 dbl-2008-hwcd (a = -1)
  EdwardsPoint double_() {
    final a = _mod(x * x);
    final b = _mod(y * y);
    final c = _mod(BigInt.two * z * z);
    final d = _mod(-a);
    final xy = _mod(x + y);
    final e = _mod(xy * xy - a - b);
    final g = _mod(d + b);
    final f = _mod(g - c);
    final h = _mod(d - b);
    return EdwardsPoint(_mod(e * f), _mod(g * h), _mod(f * g), _mod(e * h));
  }

  EdwardsPoint negate() => EdwardsPoint(_mod(-x), y, z, _mod(-t));

  /// 标量乘（double-and-add，LSB 优先），scalar 须 >= 0
  EdwardsPoint multiply(BigInt scalar) {
    if (scalar < BigInt.zero) {
      throw ArgumentError('scalar must be non-negative');
    }
    if (scalar == BigInt.zero) {
      return identity;
    }
    EdwardsPoint result = identity;
    EdwardsPoint addend = this;
    var k = scalar;
    while (k > BigInt.zero) {
      if (k.isOdd) {
        result = result.add(addend);
      }
      addend = addend.double_();
      k >>= 1;
    }
    return result;
  }

  /// 仿射坐标 (x, y)
  (BigInt, BigInt) toAffine() {
    final zi = _inv(z);
    return (_mod(x * zi), _mod(y * zi));
  }

  /// 压缩为 32 字节（LE y，最高位为 x 的最低位）
  List<int> toBytes() {
    final affine = toAffine();
    final ax = affine.$1;
    final ay = affine.$2;
    final bytes = List<int>.filled(32, 0);
    var v = ay;
    for (var i = 0; i < 32; i++) {
      bytes[i] = (v & BigInt.from(0xff)).toInt();
      v >>= 8;
    }
    if (ax.isOdd) {
      bytes[31] |= 0x80;
    }
    return bytes;
  }

  /// 从 32 字节压缩编码解压，非法编码抛 [FormatException]
  static EdwardsPoint fromBytes(List<int> bytes) {
    if (bytes.length != 32) {
      throw const FormatException('point encoding must be 32 bytes');
    }
    final sign = (bytes[31] & 0x80) != 0 ? 1 : 0;
    var y = BigInt.zero;
    for (var i = 31; i >= 0; i--) {
      final b = i == 31 ? (bytes[i] & 0x7f) : bytes[i];
      y = (y << 8) | BigInt.from(b);
    }
    if (y >= curveP) {
      throw const FormatException('y >= p');
    }
    final x = _recoverX(y, sign);
    return EdwardsPoint(x, y, BigInt.one, _mod(x * y));
  }

  bool equals(EdwardsPoint o) {
    // X1/Z1 == X2/Z2  <=>  X1*Z2 == X2*Z1（Y 同理）
    return _mod(x * o.z) == _mod(o.x * z) && _mod(y * o.z) == _mod(o.y * z);
  }
}

/// 由 y 和符号位恢复 x（RFC 8032 §5.1.3）
BigInt _recoverX(BigInt y, int sign) {
  final y2 = _mod(y * y);
  final u = _mod(y2 - BigInt.one);
  final v = _mod(curveD * y2 + BigInt.one);
  // 候选 x = u*v^3 * (u*v^7)^((p-5)/8)
  final v3 = _mod(v * v * v);
  final v7 = _mod(v3 * v3 * v);
  var xx = _mod(
      u * v3 * _mod(u * v7).modPow((curveP - BigInt.from(5)) >> 3, curveP));
  final check = _mod(v * xx * xx);
  if (check == u) {
    // xx 即为解
  } else if (check == _mod(-u)) {
    xx = _mod(xx * sqrtM1);
  } else {
    throw const FormatException('invalid point: not on curve');
  }
  if (xx == BigInt.zero && sign == 1) {
    throw const FormatException('invalid point: x=0 with sign bit');
  }
  if ((xx.isOdd ? 1 : 0) != sign) {
    xx = _mod(-xx);
  }
  return xx;
}

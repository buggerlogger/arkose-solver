// Arkose VM program - decompiled to pseudo-JavaScript
// Reconstructed by symbolic stack execution. Control flow is kept as
// labels + gotos; expressions are lifted. Verified against the runtime
// crypto trace (importKey/encrypt) and the deobfuscated opcode handlers.
//
// v_<hex> = a VM-local variable/function id.  crypto == v_e38f0.

function v_b5eec() {
  if (!(window["navigation"])) goto L0;
  if (!(window["navigation"]["entries"])) goto L2;
  v_a37db = await window["navigation"]["entries"]();
  return v_a37db["length"];
  goto L3;
  L2:
  L3:
  goto L1;
  L0:
  L1:
  return null;
}
var v_eb5be;  // = undefined (hoisted declaration)
var v_eb5be;  // = undefined (hoisted declaration)
function v_c57d6() {
  v_fe168 = window["performance"];
  if (!((!v_fe168))) goto L4;
  return [1, null];
  goto L5;
  L4:
  L5:
  v_f0a60 = v_fe168["memory"];
  if (!((!v_f0a60))) goto L6;
  return [2, null];
  goto L7;
  L6:
  L7:
  v_a6d5d = v_f0a60["jsHeapSizeLimit"];
  if (!((!v_a6d5d))) goto L8;
  return [3, null];
  goto L9;
  L8:
  L9:
  return [0, v_a6d5d];
}
var v_a0c9f;  // = undefined (hoisted declaration)
var v_a0c9f;  // = undefined (hoisted declaration)
function v_a3b1e() {
  var v_f2bf6;  // = undefined (hoisted declaration)
  if ((!v_f2bf6[4])) goto L12;  // || short-circuit, value kept
  L12:
  if (!(((!v_f2bf6[4]) || (!v_f2bf6[4]["value"])))) goto L10;
  return;
  goto L11;
  L10:
  L11:
  v_a14a1 = v_f2bf6[4];
  v_b1762 = v_b5eec();
  v_d173c = v_c57d6();
  await v_a14a1["value"]["push"]({key: "vsadsa", value: v_b1762});
  await v_a14a1["value"]["push"]({key: "basfas", value: v_d173c});
  await v_a14a1["value"]["push"]({key: "lfasdgs", value: window["arkl"]["cbid"]});
  v_e3df4 = v_a14a1["value"][25];
  v_ee667 = v_a14a1["value"][1];
  v_c7431 = v_a14a1["value"][18];
  if (!(v_e3df4)) goto L16;  // && short-circuit, value kept
  L16:
  if (!((v_e3df4 && v_ee667))) goto L15;  // && short-circuit, value kept
  L15:
  if (!(((v_e3df4 && v_ee667) && v_c7431))) goto L13;
  if (!(v_ee667["value"])) goto L21;  // && short-circuit, value kept
  L21:
  if (!((v_ee667["value"] && (v_ee667["value"]["length"] > 12)))) goto L20;  // && short-circuit, value kept
  L20:
  if (!(((v_ee667["value"] && (v_ee667["value"]["length"] > 12)) && v_c7431["value"]))) goto L19;  // && short-circuit, value kept
  L19:
  if (!((((v_ee667["value"] && (v_ee667["value"]["length"] > 12)) && v_c7431["value"]) && (v_c7431["value"]["length"] > 12)))) goto L17;
  v_a14a1["value"][25] = {key: v_e3df4["key"], value: (await v_ee667["value"]["slice"](0, 3) + await v_c7431["value"]["slice"](0, 3))};
  delete undefined;
  goto L18;
  L17:
  v_a14a1["value"][25] = {key: v_e3df4["key"], value: "abcdef"};
  delete undefined;
  L18:
  goto L14;
  L13:
  L14:
  v_cef22 = v_a14a1["value"][26];
  if (!(v_cef22)) goto L22;
  if (!(v_cef22["value"])) goto L24;
  v_a14a1["value"][26] = {key: v_cef22["key"], value: (v_cef22["value"] * 3)};
  delete undefined;
  goto L25;
  L24:
  L25:
  goto L23;
  L22:
  L23:
  v_f5428 = v_a14a1["value"][63];
  if (!(v_f5428)) goto L31;  // && short-circuit, value kept
  L31:
  if (!((v_f5428 && v_f5428["value"]))) goto L30;  // && short-circuit, value kept
  L30:
  if (!(((v_f5428 && v_f5428["value"]) && (v_f5428["key"] === "z87b89t5")))) goto L29;  // && short-circuit, value kept
  L29:
  if (!((((v_f5428 && v_f5428["value"]) && (v_f5428["key"] === "z87b89t5")) && (v_f5428["value"] > 100301)))) goto L28;  // && short-circuit, value kept
  L28:
  if (!(((((v_f5428 && v_f5428["value"]) && (v_f5428["key"] === "z87b89t5")) && (v_f5428["value"] > 100301)) && (v_f5428["value"] < 100303)))) goto L26;
  v_f5428["key"] = "mobile_sdk__dataRequestTimedOut";
  delete undefined;
  v_f5428["value"] = 1;
  delete undefined;
  goto L27;
  L26:
  L27:
}
var v_f34c0;  // = undefined (hoisted declaration)
var v_f34c0;  // = undefined (hoisted declaration)
function v_f0f8d() {
  var v_e38f0;  // = undefined (hoisted declaration)
  var v_d81df;  // = undefined (hoisted declaration)
  var v_f2bf6;  // = undefined (hoisted declaration)
  function v_ad63c() {
    return await v_e38f0["getRandomValues"](new Uint8Array(32));
  }
  function v_b14af() {
    var v_ddcd9;  // = undefined (hoisted declaration)
    return await v_e38f0["subtle"]["importKey"]("raw", v_ddcd9, {name: "AES-GCM"}, false, ["encrypt"]);
  }
  function v_da75c() {
    return await v_e38f0["getRandomValues"](new Uint8Array(12));
  }
  function v_cf74c() {
    var v_ceba8;  // = undefined (hoisted declaration)
    var v_e1fdc;  // = undefined (hoisted declaration)
    var v_ea4b4;  // = undefined (hoisted declaration)
    v_c4a91 = await window["arkl"]["aagesg"](v_ceba8);
    return await v_e38f0["subtle"]["encrypt"]({name: "AES-GCM", tagLength: 128, iv: v_e1fdc}, v_ea4b4, v_c4a91);
  }
  function v_edf4c() {
    var v_b9750;  // = undefined (hoisted declaration)
    v_a7b32 = new Uint8Array(v_b9750);
    v_fe65a = 16;
    v_ab959 = new Uint8Array(await v_a7b32["subarray"](0, (v_a7b32["length"] - v_fe65a)));
    v_e952d = new Uint8Array(await v_a7b32["subarray"]((v_a7b32["length"] - v_fe65a)));
    return {ciphertextBuffer: v_ab959["buffer"], tagBuffer: v_e952d["buffer"]};
  }
  function v_e06ca() {
    var v_e7eca;  // = undefined (hoisted declaration)
    var v_ddcd9;  // = undefined (hoisted declaration)
    v_fe97d = await window["arkl"]["basdga"](v_e7eca);
    v_b81dc = await v_e38f0["subtle"]["importKey"]("spki", v_fe97d, {name: "RSA-OAEP", hash: "SHA-256"}, false, ["encrypt"]);
    return await v_e38f0["subtle"]["encrypt"]({name: "RSA-OAEP", hash: "SHA-256"}, v_b81dc, v_ddcd9);
  }
  if (!((!v_f2bf6))) goto L32;
  return null;
  goto L33;
  L32:
  L33:
  if (!((!v_d81df))) goto L34;
  return null;
  goto L35;
  L34:
  L35:
  v_a2b51 = await JSON["stringify"](v_f2bf6);
  v_c5ba1 = v_ad63c();
  v_cae5d = await v_b14af(v_c5ba1);
  v_c81cd = await v_e06ca(v_c5ba1, v_d81df);
  v_c3bee = 0;
  L36:
  if (!((v_c3bee < v_c5ba1["length"]))) goto L37;
  v_c5ba1.i = 0;
  delete undefined;
  v_c3bee = (v_c3bee + 1);
  delete i;
  goto L36;
  L37:
  v_c28fe = v_da75c();
  v_bdb46 = await v_cf74c(v_cae5d, v_c28fe, v_a2b51);
  v_b1d50 = v_edf4c(v_bdb46);
  v_e5a90 = v_b1d50["ciphertextBuffer"];
  v_af824 = v_b1d50["tagBuffer"];
  v_f6d5d = await window["arkl"]["caasgs"](v_e5a90);
  v_cfc0a = await window["arkl"]["caasgs"](v_af824);
  v_c2275 = await window["arkl"]["caasgs"](v_c28fe["buffer"]);
  v_a64cb = await window["arkl"]["caasgs"](v_c81cd);
  return (((v_c2275 + v_cfc0a) + v_a64cb) + v_f6d5d);
}
var v_f3d8c;  // = undefined (hoisted declaration)
var v_f3d8c;  // = undefined (hoisted declaration)
function v_f37ee() {
  var v_e38f0;  // = undefined (hoisted declaration)
  var v_d81df;  // = undefined (hoisted declaration)
  var v_f2bf6;  // = undefined (hoisted declaration)
  function v_df781() {
    var v_b8f0f;  // = undefined (hoisted declaration)
    v_ce00c = [];
    v_c3bee = 0;
    L38:
    if (!((v_c3bee < v_b8f0f["length"]))) goto L39;
    v_ce00c.i = v_b8f0f.i;
    delete undefined;
    v_c3bee = (v_c3bee + 1);
    delete i;
    goto L38;
    L39:
    await v_e38f0["initPrng"](v_ce00c);
  }
  v_eddf0 = await window["msCrypto"]["getRandomValues"](new Uint8Array(48));
  v_df781(v_eddf0);
  return await v_f0f8d(v_f2bf6, v_d81df, v_e38f0);
}
var v_ca142;  // = undefined (hoisted declaration)
var v_ca142;  // = undefined (hoisted declaration)
function v_acd94() {
  var v_da4fd;  // = undefined (hoisted declaration)
  var v_f42af;  // = undefined (hoisted declaration)
  v_dd189 = await v_da4fd["util"]["decode64"](v_f42af);
  v_a9fac = await v_da4fd["asn1"]["fromDer"](v_dd189);
  return await v_da4fd["pki"]["publicKeyFromAsn1"](v_a9fac);
}
function v_b36a3() {
  var v_da4fd;  // = undefined (hoisted declaration)
  return await v_da4fd["random"]["getBytesSync"](12);
}
function v_f71fd() {
  var v_da4fd;  // = undefined (hoisted declaration)
  var v_d81df;  // = undefined (hoisted declaration)
  var v_abac2;  // = undefined (hoisted declaration)
  var v_a2b51;  // = undefined (hoisted declaration)
  v_c1030 = await v_da4fd["random"]["getBytesSync"](32);
  v_cc61f = await v_da4fd["cipher"]["createCipher"]("AES-GCM", v_c1030);
  await v_cc61f["start"]({iv: v_abac2});
  await v_cc61f["update"](await v_da4fd["util"]["createBuffer"](v_a2b51, "utf8"));
  if (!((!await v_cc61f["finish"]()))) goto L40;
  return null;
  goto L41;
  L40:
  L41:
  v_b81dc = v_acd94(v_d81df, v_da4fd);
  v_c05f5 = await v_da4fd["md"]["sha256"]["create"]();
  v_f5db5 = await v_b81dc["encrypt"](v_c1030, "RSA-OAEP", {md: v_c05f5});
  return {ciphertext: await v_cc61f["output"]["getBytes"](), tag: await v_cc61f["mode"]["tag"]["getBytes"](), encryptedSymmetricKey: v_f5db5};
}
function v_cd729() {
  var v_da4fd;  // = undefined (hoisted declaration)
  var v_d81df;  // = undefined (hoisted declaration)
  var v_f2bf6;  // = undefined (hoisted declaration)
  if (!((!v_f2bf6))) goto L42;
  return null;
  goto L43;
  L42:
  L43:
  if (!((!v_da4fd))) goto L44;
  return null;
  goto L45;
  L44:
  L45:
  if (!((!v_d81df))) goto L46;
  return null;
  goto L47;
  L46:
  L47:
  v_abac2 = v_b36a3(v_da4fd);
  v_e9083 = v_f71fd(await JSON["stringify"](v_f2bf6), v_abac2, v_d81df, v_da4fd);
  if (!((!v_e9083))) goto L48;
  return null;
  goto L49;
  L48:
  L49:
  v_f6d5d = await v_da4fd["util"]["encode64"](v_e9083["ciphertext"]);
  v_cfc0a = await v_da4fd["util"]["encode64"](v_e9083["tag"]);
  v_c2275 = await v_da4fd["util"]["encode64"](v_abac2);
  v_a64cb = await v_da4fd["util"]["encode64"](v_e9083["encryptedSymmetricKey"]);
  return (((v_c2275 + v_cfc0a) + v_a64cb) + v_f6d5d);
}
var v_a2561;  // = undefined (hoisted declaration)
var v_a2561;  // = undefined (hoisted declaration)
function v_fe02d() {
  v_e9083 = "undefined";
  v_f2bf6 = window["arkl"]["pl"];
  v_d81df = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAn7u+bMTvxBDr655ISKaHd2XvuwHNRei5qz0UZhRCigkAETlf79zgRP7BJkAv8hSPmWiLNIASnFG9RuitQlt5eD6y2USMGXB4Ac0wcjF2O99ZCypYPm/YtEYClNN9KewcFe8rSFHv30V9e+U7xIFOSnkbcE8Q9XeC3r7c4MBZeQOkvfNb3nwQ6ctwNX8Jhws06IwQdMSU+WR2FeC6mIFM+//UYfUx9G1pSVNnBw2VWrbb9DJUaeBE1TKi/Sn1wX5qKCmG3qmBzp9yeu24gOjzr9Gya14MLxYv9+fstcWBmyWT9/up0ky6CQ7Jzml3TWIqXPpAnDiZ4CXjviEtq3P2qQIDAQAB";
  v_a3b1e(v_f2bf6);
  if (!((window["arkl"]["cryptoType"] === "crypto"))) goto L50;
  v_e9083 = await v_f0f8d(v_f2bf6, v_d81df, window["arkl"]["crypto"]);
  delete result;
  goto L51;
  L50:
  if (!((window["arkl"]["cryptoType"] === "msr"))) goto L52;
  v_e9083 = v_f37ee(v_f2bf6, v_d81df, window["arkl"]["msrCrypto"]);
  delete result;
  goto L53;
  L52:
  if (!((window["arkl"]["cryptoType"] === "forge"))) goto L54;
  v_e9083 = v_cd729(v_f2bf6, v_d81df, window["arkl"]["forge"]);
  delete result;
  goto L55;
  L54:
  L55:
  L53:
  L51:
  window["arkl"]["rs"] = v_e9083;
  delete undefined;
}
v_fe02d();

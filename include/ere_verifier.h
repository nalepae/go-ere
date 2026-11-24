#ifndef ERE_VERIFIER_H
#define ERE_VERIFIER_H

#pragma once

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * Operation succeeded.
 */
#define ERE_OK 0

/**
 * A required pointer argument was null when a value was expected, or a length was above
 * `PTRDIFF_MAX`.
 */
#define ERE_ERR_NULL_PTR 1

/**
 * `zkvm_kind` was not one of the documented values.
 */
#define ERE_ERR_BAD_KIND 2

/**
 * The program verifying key bytes failed to decode.
 */
#define ERE_ERR_DECODE_PROGRAM_VK 3

/**
 * The proof bytes failed to decode.
 */
#define ERE_ERR_DECODE_PROOF 4

/**
 * The proof was well-formed but failed cryptographic verification.
 */
#define ERE_ERR_VERIFY 5

/**
 * An unexpected internal condition occurred. This indicates a bug in the
 * binding or the verifier library rather than an invalid argument.
 */
#define ERE_ERR_INTERNAL 6

/**
 * The verifier code panicked. The caller treats this status as a rejected
 * proof, because an untrusted proof or key can cause it.
 */
#define ERE_ERR_PANIC 7

/**
 * Opaque handle to a verifier bound to a program verifying key. Created by
 * [`ere_verifier_new`] and released by [`ere_verifier_free`].
 */
typedef struct EreVerifier EreVerifier;

/**
 * Constructs a verifier bound to an encoded program verifying key for the
 * selected zkVM.
 *
 * `zkvm_kind` selects the target zkVM.
 *
 * - `0` - [`zkVMKind::OpenVM`]
 * - `1` - [`zkVMKind::SP1`]
 * - `2` - [`zkVMKind::Zisk`]
 * - `3` - [`zkVMKind::LambdaVM`]
 * - `4` - [`zkVMKind::Pico`]
 *
 * On success, writes the new handle into `*output` and returns [`ERE_OK`]. The
 * caller owns the handle and must release it with [`ere_verifier_free`]. On
 * error, `*output` is set to null and the corresponding status code is
 * returned. Only a null `output` leaves `*output` unwritten.
 *
 * [`ERE_ERR_DECODE_PROGRAM_VK`] and [`ERE_ERR_PANIC`] both mean that the library rejects the key.
 *
 * # Safety
 *
 * - `encoded_program_vk_ptr` must point to `encoded_program_vk_len` readable bytes that do not
 *   change during the call. It can be null when `encoded_program_vk_len == 0`. A length above
 *   `PTRDIFF_MAX` returns [`ERE_ERR_NULL_PTR`].
 * - `output` must be a non-null, writable `*mut *mut EreVerifier`, aligned for a pointer, that
 *   does not overlap the key bytes.
 */
__attribute__((warn_unused_result))
int32_t ere_verifier_new(uint32_t zkvm_kind,
                         const uint8_t *encoded_program_vk_ptr,
                         uintptr_t encoded_program_vk_len,
                         struct EreVerifier **output);

/**
 * Verifies a proof against the verifier's program verifying key and allocates a
 * buffer holding the proven public values.
 *
 * On success, stores the buffer pointer into `*public_values_ptr` and its
 * length into `*public_values_len`, and returns [`ERE_OK`]. The caller owns the
 * buffer and must release it with [`ere_bytes_free`], passing back the exact
 * `(pointer, length)` pair. Empty public values are reported as a null pointer
 * and zero length. On error, both out-parameters are cleared to null and zero
 * and the corresponding status code is returned. Only a null `public_values_ptr` or
 * `public_values_len` leaves both unwritten.
 *
 * # Public values
 *
 * [`ERE_OK`] shows only that the program ran. The caller must compare the public values with the
 * expected bytes.
 *
 * - OpenVM and ZisK return 256 bytes, the guest output padded with zeros.
 * - LambdaVM, Pico and SP1 return the guest output.
 *
 * # Safety
 *
 * - `handle` must be a live handle returned by [`ere_verifier_new`].
 * - `encoded_proof_ptr` must point to `encoded_proof_len` readable bytes that do not change during
 *   the call. It can be null when `encoded_proof_len == 0`. A length above `PTRDIFF_MAX` returns
 *   [`ERE_ERR_NULL_PTR`].
 * - `public_values_ptr` must be a non-null, writable `*mut *mut u8`, aligned for a pointer.
 * - `public_values_len` must be a non-null, writable `*mut usize`, aligned for a `uintptr_t`.
 * - The two outputs must not overlap each other, the handle or the proof bytes.
 */
__attribute__((warn_unused_result))
int32_t ere_verifier_verify(const struct EreVerifier *handle,
                            const uint8_t *encoded_proof_ptr,
                            uintptr_t encoded_proof_len,
                            uint8_t **public_values_ptr,
                            uintptr_t *public_values_len);

/**
 * Writes the zkVM kind the verifier was constructed for into `*output` and
 * returns [`ERE_OK`]. On error, `*output` is not written.
 *
 * # Safety
 *
 * - `handle` must be a live handle returned by [`ere_verifier_new`].
 * - `output` must be a non-null, writable `*mut u32`, aligned for a `uint32_t`, that does not
 *   overlap the handle.
 */
__attribute__((warn_unused_result))
int32_t ere_verifier_zkvm_kind(const struct EreVerifier *handle,
                               uint32_t *output);

/**
 * Releases a verifier handle. The handle must not be used after this call.
 * Passing a null pointer is a no-op.
 *
 * # Safety
 *
 * `handle` must be a live handle returned by [`ere_verifier_new`] or null.
 */
void ere_verifier_free(struct EreVerifier *handle);

/**
 * Releases a byte buffer that this library allocated and handed back through a
 * `(pointer, length)` pair. The pair must be exactly what the allocating call
 * produced, and the buffer must not be used after this call. A null pointer or
 * zero length is a no-op.
 *
 * # Safety
 *
 * `ptr` and `len` must be an unmodified pair previously produced by a function
 * in this library that documents [`ere_bytes_free`] as its release routine and
 * not already freed, or `ptr` must be null.
 */
void ere_bytes_free(uint8_t *ptr, uintptr_t len);

#endif  /* ERE_VERIFIER_H */

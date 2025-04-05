// Пакет main содержит реализацию хеш-функции ГОСТ Р 34.11-94
package main

import (
	"encoding/binary" // Пакет для работы с бинарными данными
	"encoding/hex"    // Пакет для кодирования/декодирования шестнадцатеричных строк
	"fmt"             // Пакет для форматированного ввода-вывода
	"math/big"        // Пакет для работы с большими числами

	"github.com/ftomza/gogost/gost28147" // Внешний пакет для алгоритма ГОСТ 28147-89
)

const (
	BlockSize = 32 // Размер блока в байтах (256 бит)
	Size      = 32 // Размер хеш-значения в байтах (256 бит)
)

var (
	// Используем стандартные S-блоки ГОСТ Р 28147-89 как узлы замены
	SboxDefault *gost28147.Sbox = &gost28147.SboxIdGostR341194TestParamSet

	// Константы для преобразований в функции хеширования
	c2 [BlockSize]byte = [BlockSize]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	// Константа c3 используется в алгоритме для нелинейных преобразований
	c3 [BlockSize]byte = [BlockSize]byte{
		0xff, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00, 0xff,
		0xff, 0x00, 0x00, 0xff, 0x00, 0xff, 0xff, 0x00,
		0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff,
		0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00,
	}
	// Константа c4 используется в алгоритме для нелинейных преобразований
	c4 [BlockSize]byte = [BlockSize]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Константа 2^256 для вычислений контрольной суммы
	big256 *big.Int = big.NewInt(0).SetBit(big.NewInt(0), 256, 1)
)

// Hash представляет состояние хеш-функции ГОСТ Р 34.11-94
type Hash struct {
	sbox *gost28147.Sbox // S-блоки для алгоритма ГОСТ 28147-89
	size uint64          // Количество обработанных битов
	hsh  [BlockSize]byte // Текущее значение хеша
	chk  *big.Int        // Контрольная сумма
	buf  []byte          // Буфер для необработанных данных
	tmp  [BlockSize]byte // Временный буфер
}

// New создает новый экземпляр хеш-функции с указанными S-блоками
func New(sbox *gost28147.Sbox) *Hash {
	h := Hash{sbox: sbox}
	h.Reset()
	return &h
}

// Reset сбрасывает состояние хеш-функции до начального
func (h *Hash) Reset() {
	h.size = 0 // Обнуляем счетчик обработанных битов
	// Инициализируем хеш нулевым значением
	h.hsh = [BlockSize]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	h.chk = big.NewInt(0) // Обнуляем контрольную сумму
	h.buf = h.buf[:0]     // Очищаем буфер
}

// BlockSize возвращает размер блока хеш-функции в байтах
func (h *Hash) BlockSize() int {
	return BlockSize
}

// Size возвращает размер итогового хеш-значения в байтах
func (h *Hash) Size() int {
	return BlockSize
}

// fA - функция преобразования, используемая в алгоритме ГОСТ Р 34.11-94
// Выполняет XOR и циклический сдвиг блоков
func fA(in *[BlockSize]byte) *[BlockSize]byte {
	out := new([BlockSize]byte)
	// XOR последних двух блоков по 8 байт
	out[0] = in[16+0] ^ in[24+0]
	out[1] = in[16+1] ^ in[24+1]
	out[2] = in[16+2] ^ in[24+2]
	out[3] = in[16+3] ^ in[24+3]
	out[4] = in[16+4] ^ in[24+4]
	out[5] = in[16+5] ^ in[24+5]
	out[6] = in[16+6] ^ in[24+6]
	out[7] = in[16+7] ^ in[24+7]
	// Копируем первые 24 байта входных данных со смещением
	copy(out[8:], in[0:24])
	return out
}

// fP - функция перестановки, используемая в алгоритме ГОСТ Р 34.11-94
// Переставляет байты по определенному алгоритму
func fP(in *[BlockSize]byte) *[BlockSize]byte {
	return &[BlockSize]byte{
		in[0], in[8], in[16], in[24], in[1], in[9], in[17],
		in[25], in[2], in[10], in[18], in[26], in[3],
		in[11], in[19], in[27], in[4], in[12], in[20],
		in[28], in[5], in[13], in[21], in[29], in[6],
		in[14], in[22], in[30], in[7], in[15], in[23], in[31],
	}
}

// fChi - функция сжатия, используемая в алгоритме ГОСТ Р 34.11-94
// Реализует нелинейное преобразование блока
func fChi(in *[BlockSize]byte) *[BlockSize]byte {
	out := new([BlockSize]byte)
	// Вычисляем первые два байта с помощью XOR нескольких байтов из входного блока
	out[0] = in[32-2] ^ in[32-4] ^ in[32-6] ^ in[32-8] ^ in[0] ^ in[32-26]
	out[1] = in[32-1] ^ in[32-3] ^ in[32-5] ^ in[32-7] ^ in[32-31] ^ in[32-25]
	// Сдвигаем остальные байты на 2 позиции
	copy(out[2:32], in[0:30])
	return out
}

// blockReverse инвертирует порядок байтов в блоке
func blockReverse(dst, src []byte) {
	for i, j := 0, BlockSize-1; i < j; i, j = i+1, j-1 {
		dst[i], dst[j] = src[j], src[i]
	}
}

// blockXor выполняет побитовый XOR двух блоков
func blockXor(dst, a, b *[BlockSize]byte) {
	for i := 0; i < BlockSize; i++ {
		dst[i] = a[i] ^ b[i]
	}
}

// step выполняет один шаг хеш-функции ГОСТ Р 34.11-94
// hin - текущий хеш, m - обрабатываемый блок данных
func (h *Hash) step(hin, m [BlockSize]byte) [BlockSize]byte {
	out := new([BlockSize]byte)
	u := new([BlockSize]byte)
	v := new([BlockSize]byte)
	k := new([BlockSize]byte)

	// Инициализация векторов
	(*u) = hin
	(*v) = m

	// Вычисляем первый ключ шифрования
	blockXor(k, u, v)
	k = fP(k)
	blockReverse(k[:], k[:])

	// Используем алгоритм шифрования ГОСТ 28147-89 для первого блока данных
	c := gost28147.NewCipher(k[:], h.sbox)
	s := make([]byte, gost28147.BlockSize)
	c.Encrypt(s, []byte{
		hin[31], hin[30], hin[29], hin[28], hin[27], hin[26], hin[25], hin[24],
	})
	out[31] = s[0]
	out[30] = s[1]
	out[29] = s[2]
	out[28] = s[3]
	out[27] = s[4]
	out[26] = s[5]
	out[25] = s[6]
	out[24] = s[7]

	// Преобразуем u и v для второго раунда шифрования
	blockXor(u, fA(u), &c2)
	v = fA(fA(v))
	// Вычисляем второй ключ шифрования
	blockXor(k, u, v)
	k = fP(k)
	blockReverse(k[:], k[:])

	// Шифруем второй блок данных
	c = gost28147.NewCipher(k[:], h.sbox)
	c.Encrypt(s, []byte{
		hin[23], hin[22], hin[21], hin[20], hin[19], hin[18], hin[17], hin[16],
	})
	out[23] = s[0]
	out[22] = s[1]
	out[21] = s[2]
	out[20] = s[3]
	out[19] = s[4]
	out[18] = s[5]
	out[17] = s[6]
	out[16] = s[7]

	// Преобразуем u и v для третьего раунда шифрования
	blockXor(u, fA(u), &c3)
	v = fA(fA(v))
	// Вычисляем третий ключ шифрования
	blockXor(k, u, v)
	k = fP(k)
	blockReverse(k[:], k[:])

	// Шифруем третий блок данных
	c = gost28147.NewCipher(k[:], h.sbox)
	c.Encrypt(s, []byte{
		hin[15], hin[14], hin[13], hin[12], hin[11], hin[10], hin[9], hin[8],
	})
	out[15] = s[0]
	out[14] = s[1]
	out[13] = s[2]
	out[12] = s[3]
	out[11] = s[4]
	out[10] = s[5]
	out[9] = s[6]
	out[8] = s[7]

	// Преобразуем u и v для четвертого раунда шифрования
	blockXor(u, fA(u), &c4)
	v = fA(fA(v))
	// Вычисляем четвертый ключ шифрования
	blockXor(k, u, v)
	k = fP(k)
	blockReverse(k[:], k[:])

	// Шифруем четвертый блок данных
	c = gost28147.NewCipher(k[:], h.sbox)
	c.Encrypt(s, []byte{
		hin[7], hin[6], hin[5], hin[4], hin[3], hin[2], hin[1], hin[0],
	})
	out[7] = s[0]
	out[6] = s[1]
	out[5] = s[2]
	out[4] = s[3]
	out[3] = s[4]
	out[2] = s[5]
	out[1] = s[6]
	out[0] = s[7]

	// Применяем 12 раундов нелинейного преобразования fChi
	for i := 0; i < 12; i++ {
		out = fChi(out)
	}
	// Применяем XOR с входным блоком данных
	blockXor(out, out, &m)
	// Еще один раунд нелинейного преобразования
	out = fChi(out)
	// Применяем XOR с текущим значением хеша
	blockXor(out, out, &hin)
	// Применяем еще 61 раунд нелинейного преобразования
	for i := 0; i < 61; i++ {
		out = fChi(out)
	}
	return *out
}

// chkAdd добавляет байтовый блок к контрольной сумме по модулю 2^256
func (h *Hash) chkAdd(data []byte) *big.Int {
	i := big.NewInt(0).SetBytes(data)
	i.Add(i, h.chk)
	if i.Cmp(big256) != -1 {
		i.Sub(i, big256)
	}
	return i
}

// Write добавляет данные к хешируемому сообщению
// Реализует интерфейс io.Writer
func (h *Hash) Write(data []byte) (int, error) {
	h.buf = append(h.buf, data...)
	// Обрабатываем полные блоки по 32 байта
	for len(h.buf) >= BlockSize {
		h.size += BlockSize * 8                   // Увеличиваем счетчик обработанных битов
		blockReverse(h.tmp[:], h.buf[:BlockSize]) // Инвертируем порядок байтов
		h.chk = h.chkAdd(h.tmp[:])                // Обновляем контрольную сумму
		h.buf = h.buf[BlockSize:]                 // Удаляем обработанный блок из буфера
		h.hsh = h.step(h.hsh, h.tmp)              // Выполняем шаг хеширования
	}
	return len(data), nil
}

// Sum добавляет padding и возвращает итоговое хеш-значение
func (h *Hash) Sum(in []byte) []byte {
	// Сохраняем текущее состояние хеширования
	size := h.size
	chk := h.chk
	hsh := h.hsh
	block := new([BlockSize]byte)

	// Обрабатываем оставшиеся данные, если они есть
	if len(h.buf) != 0 {
		size += uint64(len(h.buf)) * 8   // Добавляем размер оставшихся данных в битах
		copy(block[:], h.buf)            // Копируем оставшиеся данные во временный блок
		blockReverse(block[:], block[:]) // Инвертируем порядок байтов
		chk = h.chkAdd(block[:])         // Обновляем контрольную сумму
		hsh = h.step(hsh, *block)        // Выполняем шаг хеширования
		block = new([BlockSize]byte)     // Сбрасываем временный блок
	}

	// Добавляем блок с размером сообщения в битах (padding)
	binary.BigEndian.PutUint64(block[24:], size)
	hsh = h.step(hsh, *block)

	// Добавляем блок с контрольной суммой
	block = new([BlockSize]byte)
	chkBytes := chk.Bytes()
	copy(block[BlockSize-len(chkBytes):], chkBytes)
	hsh = h.step(hsh, *block)

	// Инвертируем порядок байтов в итоговом хеше
	blockReverse(hsh[:], hsh[:])

	// Добавляем хеш к выходным данным
	return append(in, hsh[:]...)
}

// main демонстрирует использование хеш-функции ГОСТ Р 34.11-94
func main() {
	// Создаем новый экземпляр хеш-функции ГОСТ Р 34.11-94
	h := New(SboxDefault)

	// Данные для хеширования
	data := []byte("This is message, length=32 bytes")

	// Вычисляем хеш
	h.Write(data)
	hash := h.Sum(nil)

	// Выводим результат в шестнадцатеричном виде
	fmt.Printf("GOST Р 34.11-94 хеш: %s\n", hex.EncodeToString(hash))
}

package slice

/**
 * RemoveAtIndex: Remueve un elemento de un slice en base a su índice.

 * @param slice: Slice de cualquier tipo.
 * @param index: Índice del elemento a remover.
 */
func RemoveAtIndex[T any](slice *[]T, index int) T {
	element := (*slice)[index]
	*slice = append((*slice)[:index], (*slice)[index+1:]...)
	return element
}

/**
 * InsertAtIndex: Inserta un elemento en un slice en el índice proporcionado.

 * @param slice: Slice de cualquier tipo.
 * @param index: Índice donde se va a ingresar el elemento.
 * @param elem:  Elemento a ingresar.
 */
func InsertAtIndex[T any](slice *[]T, index int, elem T) {
	*slice = append((*slice)[:index], append([]T{elem}, (*slice)[index:]...)...)
}

/**
 * Pop: Remueve el último elemento de un slice

 * @param slice: Slice de cualquier tipo.
 * @return T: Último elemento del slice.
*/
func Pop[T any](slice *[]T) T {
	last := (*slice)[len(*slice)-1]
	*slice = (*slice)[:len(*slice)-1]
	return last
}

/**
 * Shift: Remueve el primer elemento de un slice

 * @param slice: Slice de cualquier tipo.
 * @return T: Primer elemento del slice.
 * ! Si el slice está vacío, devuelve un valor por defecto.
*/
func Shift[T any](slice *[]T) T {
	if len(*slice) == 0 {
		var zero T
		return zero
	}

	first := (*slice)[0]
	*slice = (*slice)[1:]
	return first
}

/**
 * Push: Agrega un elemento al final de un slice

 * @param slice: Slice de cualquier tipo.
 * @param elem: Elemento a agregar.
*/
func Push[T any](slice *[]T, elem T) {
	*slice = append(*slice, elem)
}